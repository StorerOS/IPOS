package condition

import (
	"fmt"
	"net"
	"net/http"
	"sort"
)

func toIPAddressFuncString(n name, key Key, values []*net.IPNet) string {
	valueStrings := []string{}
	for _, value := range values {
		valueStrings = append(valueStrings, value.String())
	}
	sort.Strings(valueStrings)

	return fmt.Sprintf("%v:%v:%v", n, key, valueStrings)
}

type ipAddressFunc struct {
	k      Key
	values []*net.IPNet
}

func (f ipAddressFunc) evaluate(values map[string][]string) bool {
	IPs := []net.IP{}
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	for _, s := range requestValue {
		IP := net.ParseIP(s)
		if IP == nil {
			panic(fmt.Errorf("invalid IP address '%v'", s))
		}

		IPs = append(IPs, IP)
	}

	for _, IP := range IPs {
		for _, IPNet := range f.values {
			if IPNet.Contains(IP) {
				return true
			}
		}
	}

	return false
}

func (f ipAddressFunc) key() Key {
	return f.k
}

func (f ipAddressFunc) name() name {
	return ipAddress
}

func (f ipAddressFunc) String() string {
	return toIPAddressFuncString(ipAddress, f.k, f.values)
}

func (f ipAddressFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	for _, value := range f.values {
		values.Add(NewStringValue(value.String()))
	}

	return map[Key]ValueSet{
		f.k: values,
	}
}

type notIPAddressFunc struct {
	ipAddressFunc
}

func (f notIPAddressFunc) evaluate(values map[string][]string) bool {
	return !f.ipAddressFunc.evaluate(values)
}

func (f notIPAddressFunc) name() name {
	return notIPAddress
}

func (f notIPAddressFunc) String() string {
	return toIPAddressFuncString(notIPAddress, f.ipAddressFunc.k, f.ipAddressFunc.values)
}

func valuesToIPNets(n name, values ValueSet) ([]*net.IPNet, error) {
	IPNets := []*net.IPNet{}
	for v := range values {
		s, err := v.GetString()
		if err != nil {
			return nil, fmt.Errorf("value %v must be string representation of CIDR for %v condition", v, n)
		}

		var IPNet *net.IPNet
		_, IPNet, err = net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("value %v must be CIDR string for %v condition", s, n)
		}

		IPNets = append(IPNets, IPNet)
	}

	return IPNets, nil
}

func newIPAddressFunc(key Key, values ValueSet) (Function, error) {
	IPNets, err := valuesToIPNets(ipAddress, values)
	if err != nil {
		return nil, err
	}

	return NewIPAddressFunc(key, IPNets...)
}

func NewIPAddressFunc(key Key, IPNets ...*net.IPNet) (Function, error) {
	if key != AWSSourceIP {
		return nil, fmt.Errorf("only %v key is allowed for %v condition", AWSSourceIP, ipAddress)
	}

	return &ipAddressFunc{key, IPNets}, nil
}

func newNotIPAddressFunc(key Key, values ValueSet) (Function, error) {
	IPNets, err := valuesToIPNets(notIPAddress, values)
	if err != nil {
		return nil, err
	}

	return NewNotIPAddressFunc(key, IPNets...)
}

func NewNotIPAddressFunc(key Key, IPNets ...*net.IPNet) (Function, error) {
	if key != AWSSourceIP {
		return nil, fmt.Errorf("only %v key is allowed for %v condition", AWSSourceIP, notIPAddress)
	}

	return &notIPAddressFunc{ipAddressFunc{key, IPNets}}, nil
}
