package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"strconv"
	"strings"
)

const (
	ErrOutOfRange     = IPv4Error("Error: Value is out of range")
	ErrInvalidSyntax  = IPv4Error("Error: Value has invalid syntax")
	ErrPrefixTooSmall = IPv4Error("Subnet prefix must be larger than the prefix")
	ErrMaskParse      = IPv4Error("Error when parsing mask")
)

type IPv4Error string

func (e IPv4Error) Error() string {
	return string(e)
}

func convertStrconvError(err error) error {
	switch {
	case errors.Is(err, strconv.ErrRange):
		return ErrOutOfRange

	case errors.Is(err, strconv.ErrSyntax):
		return ErrInvalidSyntax

	default:
		return err
	}
}

func IPv4ToInt(addrString string) (uint32, error) {
	splitAddr := strings.Split(addrString, ".")
	bytes := make([]byte, 0, 4)

	for _, octetString := range splitAddr {
		val, err := strconv.ParseUint(octetString, 10, 0)
		if val < 0 || val > 255 {
			return 0, ErrOutOfRange
		}
		if err != nil {
			errCast := err.(*strconv.NumError).Unwrap()
			err = convertStrconvError(errCast)
			return 0, err
		}

		bytes = append(bytes, byte(val))
	}

	return binary.BigEndian.Uint32(bytes), nil
}

func GetClass(address IPv4Address) string {
	ip := address.Bits()[:8]
	res, err := strconv.ParseUint(ip, 2, 0)

	if err != nil {
		return "ERROR"
	}

	switch {
	case res >= 0 && res <= 127:
		return "A"

	case res >= 128 && res <= 191:
		return "B"

	case res >= 192 && res <= 223:
		return "C"

	case res >= 224 && res <= 239:
		return "D"

	case res >= 240 && res <= 255:
		return "E"

	default:
		return "ERROR"
	}
}

type IPv4 interface {
	String() string
	Dots() string
	Bits() string
	Print()
}

type IPv4Address struct {
	Addr        uint32
	Description string
}

func (i *IPv4Address) String() string {
	return fmt.Sprintf("%d", i.Addr)
}

func (i *IPv4Address) Dots() string {
	bytes := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(bytes, i.Addr)
	return fmt.Sprintf("%v.%v.%v.%v", bytes[0], bytes[1], bytes[2], bytes[3])
}

func (i *IPv4Address) Bits() string {
	bytes := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(bytes, i.Addr)
	return fmt.Sprintf("%08b.%08b.%08b.%08b", bytes[0], bytes[1], bytes[2], bytes[3])
}

func (i *IPv4Address) Print(writer io.Writer, extended bool) {
	if extended {
		fmt.Fprintf(writer, "%s:\t%s\t%s\n", i.Description, i.Dots(), i.Bits())
		return
	}
	fmt.Fprintf(writer, "%s:\t%s\n", i.Description, i.Dots())
}

type Network struct {
	address     *IPv4Address
	mask        *IPv4Address
	network     *IPv4Address
	hostMin     *IPv4Address
	hostMax     *IPv4Address
	broadcast   *IPv4Address
	hostsPerNet *IPv4Address
}

func NewNetwork(address, mask uint32) *Network {
	ip_address := address
	ip_mask := mask
	ip_network := ip_address & ip_mask
	ip_hostMin := ip_network + 1
	ip_broadcast := ip_network + ^ip_mask
	ip_hostMax := ip_broadcast - 1
	ip_hostCount := ip_broadcast - ip_hostMin

	return &Network{
		address:     &IPv4Address{ip_address, "Address"},
		mask:        &IPv4Address{ip_mask, "Netmask"},
		network:     &IPv4Address{ip_network, "Network"},
		hostMin:     &IPv4Address{ip_hostMin, "HostMin"},
		hostMax:     &IPv4Address{ip_hostMax, "HostMax"},
		broadcast:   &IPv4Address{ip_broadcast, "Broadcast"},
		hostsPerNet: &IPv4Address{ip_hostCount, "Hosts/Net"},
	}
}

func (n *Network) Print(writer io.Writer, printDescription, extended, printClass bool) {
	if printDescription {
		n.address.Print(writer, extended)
		n.mask.Print(writer, extended)
		fmt.Fprintf(writer, "CIDR Prefix:\t/%d\n", bits.OnesCount(uint(n.mask.Addr)))
	}
	n.network.Print(writer, extended)
	if printClass {
		fmt.Fprintf(writer, "CLASS %s\n", GetClass(*n.address))
	}
	n.hostMin.Print(writer, extended)
	n.hostMax.Print(writer, extended)
	n.broadcast.Print(writer, extended)
	fmt.Fprintf(writer, "Hosts/Net:\t%d\n", n.hostsPerNet.Addr)
}

func ParseMask(mask string) (uint32, error) {
	var prefix uint32
	if len(mask) <= 1 {
		return 0, ErrMaskParse
	}

	if mask[0] == '/' {
		count, err := fmt.Sscanf(mask, "/%d", &prefix)
		if count != 1 {
			return 0, ErrMaskParse
		}
		if err != nil {
			return 0, err
		}
		return uint32(prefixToMask(prefix)), nil

	} else {
		octets := make([]byte, 4)
		count, err := fmt.Sscanf(mask, "%d.%d.%d.%d", &octets[0], &octets[1], &octets[2], &octets[3])
		if count != 4 {
			return 0, ErrMaskParse
		}
		if err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint32(octets), nil
	}
}

func prefixToMask(prefix uint32) uint32 {
	return uint32(math.Exp2(32) - math.Exp2(float64(32-prefix)))
}

func getSubnetCount(prefix, subnetPrefix int) (int, error) {
	if prefix >= subnetPrefix {
		return 0, ErrPrefixTooSmall
	}

	return int(math.Exp2(32-float64(prefix)) / math.Exp2(32-float64(subnetPrefix))), nil
}

func CreateSubnets(address, mask, subnetMask uint32) ([]Network, error) {
	subnetCount, err := getSubnetCount(bits.OnesCount32(mask), bits.OnesCount32(subnetMask))
	if err != nil {
		return nil, err
	}

	subnets := make([]Network, subnetCount)

	for i := 0; i < subnetCount; i++ {
		if i != 0 {
			address = subnets[i-1].broadcast.Addr + 1
		}

		subnets[i] = *NewNetwork(address, subnetMask)
	}
	return subnets, nil
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	extended := true

	printExtended := flag.Bool("e", false, "Display extended output")
	printClass := flag.Bool("c", false, "Display network class")
	flag.Parse()

	args := flag.Args()
	argCount := len(args)

	if argCount < 2 || argCount > 3 {
		checkError(errors.New("Need at least 2 arguments"))
	}

	address, err := IPv4ToInt(args[0])
	checkError(err)

	mask, err := ParseMask(args[1])
	checkError(err)

	network := NewNetwork(address, mask)
	network.Print(os.Stdout, true, *printExtended, *printClass)

	if argCount == 3 {
		subnetMask, err := ParseMask(args[2])
		checkError(err)

		subnets, err := CreateSubnets(address, mask, subnetMask)
		checkError(err)

		fmt.Println()

		fmt.Printf("Subnets after transition from %s to %s\n\n", args[1], args[2])
		fmt.Printf("Netmask:\t%s\n", subnets[0].mask.Dots())
		fmt.Printf("CIDR Prefix:\t%s\n", args[2])

		for i, subnet := range subnets {
			fmt.Printf("%d.\n", i+1)
			subnet.Print(os.Stdout, false, extended, false)
			fmt.Println()
		}

	}
}
