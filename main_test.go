package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPConversion(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		expected    uint32
		expectedErr error
	}{
		{
			desc:        "Success",
			input:       "192.168.0.1",
			expected:    3232235521,
			expectedErr: nil,
		},
		{
			desc:        "Negative number fails",
			input:       "-4",
			expected:    0,
			expectedErr: ErrInvalidSyntax,
		},
		{
			desc:        "Empty input fails",
			input:       "",
			expected:    0,
			expectedErr: ErrInvalidSyntax,
		},
		{
			desc:        "Out of range",
			input:       "256.168.0.1",
			expected:    0,
			expectedErr: ErrOutOfRange,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actual, actualErr := IPv4ToInt(tC.input)

			assert.Equal(t, tC.expectedErr, actualErr)
			assert.Equal(t, tC.expected, actual)
		})
	}
}

func TestIPv4(t *testing.T) {
	t.Run("Bits conversion", func(t *testing.T) {
		input := uint32(3232235521)
		ip := IPv4Address{Addr: input}

		expected := "11000000.10101000.00000000.00000001"
		actual := ip.Bits()

		assert.Equal(t, expected, actual)
	})

	t.Run("Dots conversion", func(t *testing.T) {
		input := uint32(3232235521)
		ip := IPv4Address{Addr: input}

		expected := "192.168.0.1"
		actual := ip.Dots()

		assert.Equal(t, expected, actual)
	})

	t.Run("Print short", func(t *testing.T) {
		input := uint32(3232235521)
		ip := IPv4Address{Addr: input, Description: "test desc"}

		buf := bytes.Buffer{}
		ip.Print(&buf, false)

		expected := "test desc:\t192.168.0.1\n"
		actual := buf.String()

		assert.Equal(t, expected, actual)
	})

	t.Run("Print extended", func(t *testing.T) {
		input := uint32(3232235521)
		ip := IPv4Address{Addr: input, Description: "test desc"}

		buf := bytes.Buffer{}
		ip.Print(&buf, true)

		expected := "test desc:\t192.168.0.1\t11000000.10101000.00000000.00000001\n"
		actual := buf.String()

		assert.Equal(t, expected, actual)
	})
}

func TestSubnets(t *testing.T) {
	testCases := []struct {
		desc        string
		address     uint32
		mask        uint32
		subnetMask  uint32
		expectedErr error
		expected    []Network
	}{
		{
			desc:        "Subnet prefix too small error",
			address:     3232235521,
			mask:        4294967040,
			subnetMask:  4294967040,
			expectedErr: ErrPrefixTooSmall,
			expected:    nil,
		},
		{
			desc:        "Correct",
			address:     3232235521,
			mask:        4294967040,
			subnetMask:  4294967168,
			expectedErr: nil,
			expected: []Network{
				{
					address:     &IPv4Address{3232235521, "Address"},
					mask:        &IPv4Address{4294967168, "Netmask"},
					network:     &IPv4Address{3232235520, "Network"},
					hostMin:     &IPv4Address{3232235521, "HostMin"},
					hostMax:     &IPv4Address{3232235646, "HostMax"},
					broadcast:   &IPv4Address{3232235647, "Broadcast"},
					hostsPerNet: &IPv4Address{126, "Hosts/Net"},
				},
				{
					address:     &IPv4Address{3232235648, "Address"},
					mask:        &IPv4Address{4294967168, "Netmask"},
					network:     &IPv4Address{3232235648, "Network"},
					hostMin:     &IPv4Address{3232235649, "HostMin"},
					hostMax:     &IPv4Address{3232235774, "HostMax"},
					broadcast:   &IPv4Address{3232235775, "Broadcast"},
					hostsPerNet: &IPv4Address{126, "Hosts/Net"},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actual, actualErr := CreateSubnets(tC.address, tC.mask, tC.subnetMask)

			assert.Equal(t, tC.expectedErr, actualErr)
			assert.Equal(t, tC.expected, actual)
		})
	}
}

func TestIPv4Parsing(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		expected    uint32
		expectedErr error
	}{
		{
			desc:        "/xx mask",
			input:       "/25",
			expected:    4294967168,
			expectedErr: nil,
		},
		{
			desc:        "x.x.x.x mask",
			input:       "255.255.255.128",
			expected:    4294967168,
			expectedErr: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actual, actualErr := ParseMask(tC.input)

			assert.Equal(t, tC.expectedErr, actualErr)
			assert.Equal(t, tC.expected, actual)
		})
	}
}
