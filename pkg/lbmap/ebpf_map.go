package lbmap

import (
	"net"

	ebpf "github.com/cilium/ebpf"
	"github.com/pkg/errors"
)

const (
	maxEntries = 512
)

var (
	// ErrNoLoadPinnedMap represent pinned map isn't be loaded from userspace
	ErrNoLoadPinnedMap error = errors.New("load pinned map from userspace before you use")

	// ErrMapAlreadyLoaded represent a pinned map has already loaded, it can't be loaded twice
	ErrMapAlreadyLoaded error = errors.New("Map already loaded")
)

type (
	// LoadBalanceMap is a bpf map for golang to accessing
	LoadBalanceMap struct {
		SourceAddr uint32
		DestAddr   uint32
		Bytes      uint64
		Packages   uint64
		Mac        [6]uint8
		IfIndex    uint16
	}

	// BackendServer represent the backend server for loadbalancer
	BackendServer struct {
		SourceAddr string
		DestAddr   string
		Mac        string
		Ifindex    uint16
	}

	// LoadBalanceBPFMapper provides methods to operation bpf mapper of xdp_lb
	LoadBalanceBPFMapper interface {
		Get() ([]LoadBalanceMap, error)
		Set(servers []BackendServer) error
		Load(name string) error
	}

	bpfMapper struct {
		name   string
		bpfMap *ebpf.Map
	}
)

// New create a LoadBalanceBPFMapper object
func New() LoadBalanceBPFMapper {
	return &bpfMapper{}
}

func (m *bpfMapper) Load(name string) (err error) {
	if m.bpfMap != nil {
		return ErrMapAlreadyLoaded
	}

	m.bpfMap, err = ebpf.LoadPinnedMap(name)
	if err != nil {
		return errors.Wrapf(err, "Load pinned map %s", name)
	}
	return nil
}

func (m *bpfMapper) Get() ([]LoadBalanceMap, error) {
	if m.bpfMap == nil {
		return nil, ErrNoLoadPinnedMap
	}
	var i uint32 = 0
	var results []LoadBalanceMap
	for ; i < maxEntries; i++ {
		var lb LoadBalanceMap
		err := m.bpfMap.Lookup(i, &lb)
		if err != nil {
			return nil, errors.Wrapf(err, "lookup map of key %d", i)
		}

		results = append(results, lb)
	}
	return results, nil
}

func (m *bpfMapper) Set(servers []BackendServer) error {
	if m.bpfMap == nil {
		return ErrNoLoadPinnedMap
	}
	var serversNum uint32 = uint32(len(servers))
	if serversNum == 0 {
		return errors.New("servers can't be empty")
	}
	var i uint32 = 0
	for ; i < maxEntries; i++ {
		j := i % serversNum
		daddr := InetAton(servers[j].DestAddr)
		saddr := InetAton(servers[j].SourceAddr)
		mac, err := net.ParseMAC(servers[j].Mac)
		if err != nil {
			return errors.Wrapf(err, "Invalid mac %s address, convert error", servers[j].Mac)
		}

		var lb LoadBalanceMap = LoadBalanceMap{
			SourceAddr: saddr,
			DestAddr:   daddr,
			Bytes:      0,
			Packages:   0,
			IfIndex:    servers[j].Ifindex,
		}
		for i, m := range mac {
			lb.Mac[i] = m
		}
		err = m.bpfMap.Update(i, lb, ebpf.UpdateAny)
		if err != nil {
			return errors.Wrapf(err, "update key %d , value %+v", i, lb)
		}
	}
	return nil
}
