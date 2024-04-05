package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
	"strconv"
	"strings"
)

type IPv4 interface {
	String() string
	UInt32() uint32
	Dots() string
	Bits() string
	Print(extended bool)
}

type IPv4Type struct {
	uint32      uint32
	description string
}

func (t *IPv4Type) String() string {
	return strconv.FormatUint(uint64(t.uint32), 10)
}

func (t *IPv4Type) UInt32() uint32 {
	return uint32(t.uint32)
}

func (t *IPv4Type) Dots() string {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, t.UInt32())
	return fmt.Sprintf("%v.%v.%v.%v", bytes[0], bytes[1], bytes[2], bytes[3])
}

func (t *IPv4Type) Bits() string {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, t.UInt32())
	return fmt.Sprintf("%08b.%08b.%08b.%08b", bytes[0], bytes[1], bytes[2], bytes[3])
}

func (t *IPv4Type) Print(extended bool) {
	if extended {
		fmt.Printf("%s:\t%s\t%s\n", t.description, t.Dots(), t.Bits())
		return
	}
	fmt.Printf("%s:\t%s\n", t.description, t.Dots())
}

type Network struct {
	address     IPv4Type
	mask        IPv4Type
	network     IPv4Type
	hostMin     IPv4Type
	hostMax     IPv4Type
	broadcast   IPv4Type
	hostsPerNet IPv4Type
}

func StringToUInt32(string string) (uint32, error) {
	num, err := strconv.ParseUint(string, 10, 32)
	return uint32(num), err
}

func NewNetwork(address, mask string) (*Network, error) {
	// split the leading '/'
	mask = mask[1:]
	ip_mask, err := StringToUInt32(mask)
	if err != nil {
		return nil, err
	}

	ip_address, err := IPToInt(address)
	ip_mask = uint32(math.Exp2(32) - math.Exp2(32-float64(ip_mask)))
	ip_network := ip_address & ip_mask
	ip_hostMin := ip_network + 1
	ip_broadcast := ip_network + ^ip_mask
	ip_hostMax := ip_broadcast - 1
	ip_hostCount := ip_broadcast - ip_hostMin

	return &Network{
		address:     IPv4Type{ip_address, "Address"},
		mask:        IPv4Type{ip_mask, "Netmask"},
		network:     IPv4Type{ip_network, "Network"},
		hostMin:     IPv4Type{ip_hostMin, "HostMin"},
		hostMax:     IPv4Type{ip_hostMax, "HostMax"},
		broadcast:   IPv4Type{ip_broadcast, "Broadcast"},
		hostsPerNet: IPv4Type{ip_hostCount, "Hosts/Net"},
	}, nil
}

func NewNetworkFromInt(address, mask uint32) *Network {
	ip_address := address
	ip_mask := mask
	ip_network := ip_address & ip_mask
	ip_hostMin := ip_network + 1
	ip_broadcast := ip_network + ^ip_mask
	ip_hostMax := ip_broadcast - 1
	ip_hostCount := ip_broadcast - ip_hostMin

	return &Network{
		address:     IPv4Type{ip_address, "Address"},
		mask:        IPv4Type{ip_mask, "Netmask"},
		network:     IPv4Type{ip_network, "Network"},
		hostMin:     IPv4Type{ip_hostMin, "HostMin"},
		hostMax:     IPv4Type{ip_hostMax, "HostMax"},
		broadcast:   IPv4Type{ip_broadcast, "Broadcast"},
		hostsPerNet: IPv4Type{ip_hostCount, "Hosts/Net"},
	}
}

func PrintNetwork(network *Network, printDescription bool, extended bool) {
	if printDescription {
		network.address.Print(extended)
		network.mask.Print(extended)
		fmt.Printf("CIDR Prefix:\t/%d\n", bits.OnesCount(uint(network.mask.UInt32())))
	}
	network.network.Print(extended)
	network.hostMin.Print(extended)
	network.hostMax.Print(extended)
	network.broadcast.Print(extended)
	fmt.Printf("Hosts/Net:\t%d\n", network.hostsPerNet.UInt32())
}

func Subnets(address, mask, subnetMask string) ([]Network, error) {
	var prefix float64
	var subnetPrefix float64

	count, err := fmt.Sscanf(mask, "/%f", &prefix)
	if count != 1 {
		return nil, errors.New("Error when adding network")
	}
	if err != nil {
		return nil, err
	}

	count, err = fmt.Sscanf(subnetMask, "/%f", &subnetPrefix)
	if count != 1 {
		return nil, errors.New("Error when adding subnet")
	}
	if err != nil {
		return nil, err
	}

	if prefix >= subnetPrefix {
		return nil, errors.New("Subnet prefix must be larger than the prefix")
	}

	subnetCount := int(math.Exp2(32-prefix) / math.Exp2(32-subnetPrefix))

	subnets := make([]Network, subnetCount)

	for i := 0; i < subnetCount; i++ {
		var ip uint32
		var err error

		if i == 0 {
			ip, err = IPToInt(address)
			if err != nil {
				return nil, err
			}
		} else {
			ip = subnets[i-1].broadcast.uint32 + 1
		}

		subnetMaskInt := uint32(math.Exp2(32) - math.Exp2(32-subnetPrefix))

		subnets[i] = *NewNetworkFromInt(ip, subnetMaskInt)
	}
	return subnets, nil
}

func IPToInt(thing string) (uint32, error) {
	str := strings.Split(thing, ".")
	var bytes []byte

	for _, part := range str {
		val, err := strconv.ParseInt(part, 10, 0)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		bytes = append(bytes, byte(val))
	}

	return binary.BigEndian.Uint32(bytes), nil
}

func main() {
	args := os.Args[1:]
	argCount := len(args)
	if argCount < 2 || argCount > 3 {
		fmt.Println("Wrong number of input arguments")
		os.Exit(1)
	}

	extended := true

	if argCount == 2 {
		network, err := NewNetwork(args[0], args[1])
		if err != nil {
			fmt.Println("Error when adding network")
			os.Exit(1)
		}
		PrintNetwork(network, true, extended)
	}

	if argCount == 3 {
		network, err := NewNetwork(args[0], args[1])
		if err != nil {
			fmt.Println("Error when adding subnet")
			os.Exit(1)
		}
		subnets, err := Subnets(args[0], args[1], args[2])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		PrintNetwork(network, true, extended)
		fmt.Println("")

		fmt.Printf("Subnets after transition from %s to %s\n\n", args[1], args[2])
		fmt.Printf("Netmask:\t%s\n", subnets[0].mask.Dots())
		fmt.Printf("CIDR Prefix:\t%s\n", args[2])

		for i, subnet := range subnets {
			fmt.Printf("\n")
			fmt.Printf("%d.\n", i+1)
			PrintNetwork(&subnet, false, extended)
		}
	}
}
