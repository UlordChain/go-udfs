package config

import (
	"errors"
	"fmt"

	iaddr "gx/ipfs/QmZc5PLgxW61uTPG24TroxHDF6xzgbhZZQf5i53ciQC47Y/go-ipfs-addr"
	// Needs to be imported so that users can import this package directly
	// and still parse the bootstrap addresses.
	_ "gx/ipfs/QmeHJXPqCNzSFbVkYM1uQLuM2L5FyJB9zukQ7EeqRP8ZC9/go-multiaddr-dns"
)

// DefaultBootstrapAddresses are the hardcoded bootstrap addresses
// for IPFS. they are nodes run by the IPFS team. docs on these later.
// As with all p2p networks, bootstrap is an important security concern.
//
// NOTE: This is here -- and not inside cmd/ipfs/init.go -- because of an
// import dependency issue. TODO: move this into a config/default/ package.
var DefaultBootstrapAddresses = []string{
	"/dns4/bootstrap0.udfs.one/tcp/4001/ipfs/QmZwfRydFZrL9ARqpqKLjP8b8EYQcDPYD6DxyxKmxbYGsd",
	"/dns4/bootstrap1.udfs.one/tcp/4001/ipfs/QmYtMoTxTMCETEwJgB3JAz8Ysx8D9pScH9HojPCMK6KPwz",
	"/dns4/bootstrap2.udfs.one/tcp/4001/ipfs/QmaV2SD8BrDv8FRjd9cqUXA5W7DA4EjH9VKqbkFYBVygPM",
	"/dns4/bootstrap3.udfs.one/tcp/4001/ipfs/QmX58rWnvtRVV16evApDzSkYvboz19M5zMTVNBtQPssiGL",
	//"/dns4/bootstrap0.ulord.one/tcp/4001/ipfs/QmbETUnWes7zdwZkkMGgPRtpZAYpFPxrUrCYy7fWi7JjFY",
	//"/dns4/bootstrap1.ulord.one/tcp/4001/ipfs/QmUEBGtsPNLyngqfqTtGEnR5FBVn4Nkf2Zj7PvcuaYRKxA",
	//"/dns4/bootstrap2.ulord.one/tcp/4001/ipfs/QmWCL8zrsrqe2XibqKDEdgfbXZnLwtZRSK4Ww11nYi3oq4",
	//"/dns4/bootstrap3.ulord.one/tcp/4001/ipfs/QmRq3cTbMHwNXuWEULsqrKUAYbswZv9xQKDeyPyQSnf3yY",
	//"/dns4/bootstrap4.ulord.one/tcp/4001/ipfs/Qmb5ATRjRLBfhJYCS3WpuaYgbyviSPuh3H9p9oNgrq2m78",
	//"/dns4/bootstrap5.ulord.one/tcp/4001/ipfs/QmYhJv1f6uShrfy3SerzDmyn5VWXxPWp6DgMxmzMEfhX4Y",
	//"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	//"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	//"/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	//"/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	//"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",            // mars.i.udfs.io
	//"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",           // pluto.i.udfs.io
	//"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",           // saturn.i.udfs.io
	//"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",             // venus.i.udfs.io
	//"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",            // earth.i.udfs.io
	//"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",  // pluto.i.udfs.io
	//"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",  // saturn.i.udfs.io
	//"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64", // venus.i.udfs.io
	//"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd", // earth.i.udfs.io
}

// BootstrapPeer is a peer used to bootstrap the network.
type BootstrapPeer iaddr.IPFSAddr

// ErrInvalidPeerAddr signals an address is not a valid peer address.
var ErrInvalidPeerAddr = errors.New("invalid peer address")

func (c *Config) BootstrapPeers() ([]BootstrapPeer, error) {
	return ParseBootstrapPeers(c.Bootstrap)
}

// DefaultBootstrapPeers returns the (parsed) set of default bootstrap peers.
// if it fails, it returns a meaningful error for the user.
// This is here (and not inside cmd/ipfs/init) because of module dependency problems.
func DefaultBootstrapPeers() ([]BootstrapPeer, error) {
	ps, err := ParseBootstrapPeers(DefaultBootstrapAddresses)
	if err != nil {
		return nil, fmt.Errorf(`failed to parse hardcoded bootstrap peers: %s
This is a problem with the ipfs codebase. Please report it to the dev team.`, err)
	}
	return ps, nil
}

func (c *Config) SetBootstrapPeers(bps []BootstrapPeer) {
	c.Bootstrap = BootstrapPeerStrings(bps)
}

func ParseBootstrapPeer(addr string) (BootstrapPeer, error) {
	ia, err := iaddr.ParseString(addr)
	if err != nil {
		return nil, err
	}
	return BootstrapPeer(ia), err
}

func ParseBootstrapPeers(addrs []string) ([]BootstrapPeer, error) {
	peers := make([]BootstrapPeer, len(addrs))
	var err error
	for i, addr := range addrs {
		peers[i], err = ParseBootstrapPeer(addr)
		if err != nil {
			return nil, err
		}
	}
	return peers, nil
}

func BootstrapPeerStrings(bps []BootstrapPeer) []string {
	bpss := make([]string, len(bps))
	for i, p := range bps {
		bpss[i] = p.String()
	}
	return bpss
}
