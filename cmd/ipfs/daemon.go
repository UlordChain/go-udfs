package main

import (
	"bytes"
	"context"
	"encoding/json"
	_ "expvar"
	"fmt"
	"gx/ipfs/QmPEpj17FDRpc7K1aArKZp3RsHtzRMKykeK9GVgn4WQGPR/go-ipfs-config"
	"gx/ipfs/QmUDTcnDp2WssbmiDLC6aYurUeyt7QeRakHUQMxA2mZ5iB/go-libp2p/p2p/protocol/verify"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	version "github.com/ipfs/go-ipfs"
	utilmain "github.com/ipfs/go-ipfs/cmd/ipfs/util"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/commands"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/core/corerepo"
	nodeMount "github.com/ipfs/go-ipfs/fuse/node"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	migrate "github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
	"github.com/ipfs/go-ipfs/udfs/ca"
	"github.com/pkg/errors"

	"gx/ipfs/QmNkxFCmPtr2RQxjZNRCNryLud4L9wMEiBJsLgF14MqTHj/go-bitswap"
	mprome "gx/ipfs/QmQXBfkuwgMaPx334WuL9NmyrKnbZ5udaWnHTHEsts2x3T/go-metrics-prometheus"
	"gx/ipfs/QmSXUokcP4TJpFfqozT69AVAYRtzXVMUjzQVkYX41R9Svs/go-ipfs-cmds"
	ma "gx/ipfs/QmT4U94DnD8FRfqr21obWY32HLM5VExccPKMjQHofeYqr9/go-multiaddr"
	"gx/ipfs/QmTQuFQWHAWy4wMH6ZyPfGiawA5u9T8rs79FENoV8yXaoS/client_golang/prometheus"
	"gx/ipfs/Qmaabb1tJZ2CX5cp6MuuiGgns71NYoxdgQP6Xdid1dVceC/go-multiaddr-net"
	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

const (
	adjustFDLimitKwd          = "manage-fdlimit"
	enableGCKwd               = "enable-gc"
	initOptionKwd             = "init"
	initProfileOptionKwd      = "init-profile"
	ipfsMountKwd              = "mount-ipfs"
	ipnsMountKwd              = "mount-ipns"
	migrateKwd                = "migrate"
	mountKwd                  = "mount"
	offlineKwd                = "offline"
	routingOptionKwd          = "routing"
	routingOptionSupernodeKwd = "supernode"
	routingOptionDHTClientKwd = "dhtclient"
	routingOptionDHTKwd       = "dht"
	routingOptionNoneKwd      = "none"
	routingOptionDefaultKwd   = "default"
	unencryptTransportKwd     = "disable-transport-encryption"
	unrestrictedApiAccessKwd  = "unrestricted-api"
	writableKwd               = "writable"
	enablePubSubKwd           = "enable-pubsub-experiment"
	enableIPNSPubSubKwd       = "enable-namesys-pubsub"
	enableMultiplexKwd        = "enable-mplex-experiment"
	verifyTxid                = "txid"
	verifyVoutid              = "voutid"
	verifySecret              = "secret"
	reportUosAccount = "account"

	// apiAddrKwd    = "address-api"
	// swarmAddrKwd  = "address-swarm"
)

var daemonCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Run a network-connected IPFS node.",
		ShortDescription: `
'ipfs daemon' runs a persistent ipfs daemon that can serve commands
over the network. Most applications that use IPFS will do so by
communicating with a daemon over the HTTP API. While the daemon is
running, calls to 'ipfs' commands will be sent over the network to
the daemon.
`,
		LongDescription: `
The daemon will start listening on ports on the network, which are
documented in (and can be modified through) 'ipfs config Addresses'.
For example, to change the 'Gateway' port:

  ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8082

The API address can be changed the same way:

  udfs config Addresses.API /ip4/127.0.0.1/tcp/5002

Make sure to restart the daemon after changing addresses.

By default, the gateway is only accessible locally. To expose it to
other computers in the network, use 0.0.0.0 as the ip address:

  udfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080

Be careful if you expose the API. It is a security risk, as anyone could
control your node remotely. If you need to control the node remotely,
make sure to protect the port as you would other services or database
(firewall, authenticated proxy, etc).

HTTP Headers

ipfs supports passing arbitrary headers to the API and Gateway. You can
do this by setting headers on the API.HTTPHeaders and Gateway.HTTPHeaders
keys:

  udfs config --json API.HTTPHeaders.X-Special-Header '["so special :)"]'
  udfs config --json Gateway.HTTPHeaders.X-Special-Header '["so special :)"]'

Note that the value of the keys is an _array_ of strings. This is because
headers can have more than one value, and it is convenient to pass through
to other libraries.

CORS Headers (for API)

You can setup CORS headers the same way:

  udfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["example.com"]'
  udfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "GET", "POST"]'
  udfs config --json API.HTTPHeaders.Access-Control-Allow-Credentials '["true"]'

Shutdown

To shutdown the daemon, send a SIGINT signal to it (e.g. by pressing 'Ctrl-C')
or send a SIGTERM signal to it (e.g. with 'kill'). It may take a while for the
daemon to shutdown gracefully, but it can be killed forcibly by sending a
second signal.

UDFS_PATH environment variable

ipfs uses a repository in the local file system. By default, the repo is
located at ~/.udfs. To change the repo location, set the $UDFS_PATH
environment variable:

  export UDFS_PATH=/path/to/ipfsrepo

Routing

IPFS by default will use a DHT for content routing. There is a highly
experimental alternative that operates the DHT in a 'client only' mode that
can be enabled by running the daemon as:

  udfs daemon --routing=dhtclient

This will later be transitioned into a config option once it gets out of the
'experimental' stage.

DEPRECATION NOTICE

Previously, udfs used an environment variable as seen below:

  export API_ORIGIN="http://localhost:8888/"

This is deprecated. It is still honored in this version, but will be removed
in a future version, along with this notice. Please move to setting the HTTP
Headers.
`,
	},

	Options: []cmdkit.Option{
		cmdkit.BoolOption(initOptionKwd, "Initialize udfs with default settings if not already initialized"),
		cmdkit.StringOption(initProfileOptionKwd, "Configuration profiles to apply for --init. See udfs init --help for more"),
		cmdkit.StringOption(routingOptionKwd, "Overrides the routing option").WithDefault(routingOptionDefaultKwd),
		cmdkit.BoolOption(mountKwd, "Mounts IPFS to the filesystem"),
		cmdkit.BoolOption(writableKwd, "Enable writing objects (with POST, PUT and DELETE)"),
		cmdkit.StringOption(ipfsMountKwd, "Path to the mountpoint for IPFS (if using --mount). Defaults to config setting."),
		cmdkit.StringOption(ipnsMountKwd, "Path to the mountpoint for IPNS (if using --mount). Defaults to config setting."),
		cmdkit.BoolOption(unrestrictedApiAccessKwd, "Allow API access to unlisted hashes"),
		cmdkit.BoolOption(unencryptTransportKwd, "Disable transport encryption (for debugging protocols)"),
		cmdkit.BoolOption(enableGCKwd, "Enable automatic periodic repo garbage collection"),
		cmdkit.BoolOption(adjustFDLimitKwd, "Check and raise file descriptor limits if needed").WithDefault(true),
		cmdkit.BoolOption(offlineKwd, "Run offline. Do not connect to the rest of the network but provide local API."),
		cmdkit.BoolOption(migrateKwd, "If true, assume yes at the migrate prompt. If false, assume no."),
		cmdkit.BoolOption(enablePubSubKwd, "Instantiate the udfs daemon with the experimental pubsub feature enabled."),
		cmdkit.BoolOption(enableIPNSPubSubKwd, "Enable IPNS record distribution through pubsub; enables pubsub."),
		cmdkit.BoolOption(enableMultiplexKwd, "Add the experimental 'go-multiplex' stream muxer to libp2p on construction.").WithDefault(true),

		cmdkit.StringOption(verifyTxid, "Set the verify txid, NOTE: it will save to config."),
		cmdkit.StringOption(verifySecret, "Set the verify sercret, NOTE: it will save to config."),
		cmdkit.IntOption(verifyVoutid, "Set the verify voutid, NOTE: it will save to config.").WithDefault(-1),
		cmdkit.StringOption(reportUosAccount, "Set the uos account, it will be used to got reward. NOTE: it will save to config."),

		// TODO: add way to override addresses. tricky part: updating the config if also --init.
		// cmdkit.StringOption(apiAddrKwd, "Address for the daemon rpc API (overrides config)"),
		// cmdkit.StringOption(swarmAddrKwd, "Address for the swarm socket (overrides config)"),
	},
	Subcommands: map[string]*cmds.Command{},
	Run:         daemonFunc,
}

// defaultMux tells mux to serve path using the default muxer. This is
// mostly useful to hook up things that register in the default muxer,
// and don't provide a convenient http.Handler entry point, such as
// expvar and http/pprof.
func defaultMux(path string) corehttp.ServeOption {
	return func(node *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.Handle(path, http.DefaultServeMux)
		return mux, nil
	}
}

func daemonFunc(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	// Inject metrics before we do anything
	err := mprome.Inject()
	if err != nil {
		log.Errorf("Injecting prometheus handler for metrics failed with message: %s\n", err.Error())
	}

	// let the user know we're going.
	fmt.Printf("Initializing daemon...\n")

	// print the udfs version
	printVersion()

	managefd, _ := req.Options[adjustFDLimitKwd].(bool)
	if managefd {
		if changedFds, newFdsLimit, err := utilmain.ManageFdLimit(); err != nil {
			log.Errorf("setting file descriptor limit: %s", err)
		} else {
			if changedFds {
				fmt.Printf("Successfully raised file descriptor limit to %d.\n", newFdsLimit)
			}
		}
	}

	cctx := env.(*oldcmds.Context)

	go func() {
		<-req.Context.Done()
		fmt.Println("Received interrupt signal, shutting down...")
		fmt.Println("(Hit ctrl-c again to force-shutdown the daemon.)")
	}()

	// check transport encryption flag.
	unencrypted, _ := req.Options[unencryptTransportKwd].(bool)
	if unencrypted {
		log.Warningf(`Running with --%s: All connections are UNENCRYPTED.
		You will not be able to connect to regular encrypted networks.`, unencryptTransportKwd)
	}

	// first, whether user has provided the initialization flag. we may be
	// running in an uninitialized state.
	initialize, _ := req.Options[initOptionKwd].(bool)
	if initialize {

		cfg := cctx.ConfigRoot
		if !fsrepo.IsInitialized(cfg) {
			profiles, _ := req.Options[initProfileOptionKwd].(string)

			err := initWithDefaults(os.Stdout, cfg, profiles)
			if err != nil {
				return err
			}
		}
	}

	// acquire the repo lock _before_ constructing a node. we need to make
	// sure we are permitted to access the resources (datastore, etc.)
	repo, err := fsrepo.Open(cctx.ConfigRoot)
	switch err {
	default:
		return err
	case fsrepo.ErrNeedMigration:
		domigrate, found := req.Options[migrateKwd].(bool)
		fmt.Println("Found outdated fs-repo, migrations need to be run.")

		if !found {
			domigrate = YesNoPrompt("Run migrations now? [y/N]")
		}

		if !domigrate {
			fmt.Println("Not running migrations of fs-repo now.")
			fmt.Println("Please get fs-repo-migrations from https://dist.udfs.io")
			return fmt.Errorf("fs-repo requires migration")
		}

		err = migrate.RunMigration(fsrepo.RepoVersion)
		if err != nil {
			fmt.Println("The migrations of fs-repo failed:")
			fmt.Printf("  %s\n", err)
			fmt.Println("If you think this is a bug, please file an issue and include this whole log output.")
			fmt.Println("  https://github.com/ipfs/fs-repo-migrations")
			return err
		}

		repo, err = fsrepo.Open(cctx.ConfigRoot)
		if err != nil {
			return err
		}
	case nil:
		break
	}

	cfg, err := cctx.GetConfig()
	if err != nil {
		return err
	}

	offline, _ := req.Options[offlineKwd].(bool)
	ipnsps, _ := req.Options[enableIPNSPubSubKwd].(bool)
	pubsub, _ := req.Options[enablePubSubKwd].(bool)
	mplex, _ := req.Options[enableMultiplexKwd].(bool)

	txid, _ := req.Options[verifyTxid].(string)
	secret, _ := req.Options[verifySecret].(string)
	voutid, _ := req.Options[verifyVoutid].(int)
	account, _ := req.Options[reportUosAccount].(string)

	rcfg, err := repo.Config()
	if err != nil {
		return err
	}

	needSave := false
	if txid != "" {
		rcfg.Verify.Txid = txid
		needSave = true
	}
	if voutid != -1 {
		rcfg.Verify.Voutid = int32(voutid)
		needSave = true
	}
	if secret != "" {
		rcfg.Verify.Secret = secret
		needSave = true
	}
	if account != "" {
		rcfg.Report.Account = account
		needSave = true
	}

	if needSave {
		err = repo.SetConfig(rcfg)
		if err != nil {
			return errors.Wrap(err, "Save verify info to config file failed")
		}
	}

	if !offline {
		// check verify config
		err = verify.CheckVerifyInfo(&rcfg.Verify)
		if err != nil {
			return errors.Wrap(err, "check verify info failed")
		}

		err = verify.CheckUCenterInfo(&rcfg.UCenter)
		if err != nil {
			return errors.Wrap(err, "check ucenter info failed")
		}

		if rcfg.Report.Account == "" {
			return errors.New("must provide uos account from config or option of command daemon")
		}

		// make sure the license valid
		err = doVerify(repo, rcfg)
		if err != nil {
			return err
		}
	}

	// Start assembling node config
	ncfg := &core.BuildCfg{
		Repo:                        repo,
		Permanent:                   true, // It is temporary way to signify that node is permanent
		Online:                      !offline,
		DisableEncryptedConnections: unencrypted,
		ExtraOpts: map[string]bool{
			"pubsub": pubsub,
			"ipnsps": ipnsps,
			"mplex":  mplex,
		},
		//TODO(Kubuxu): refactor Online vs Offline by adding Permanent vs Ephemeral
	}

	routingOption, _ := req.Options[routingOptionKwd].(string)
	if routingOption == routingOptionDefaultKwd {
		cfg, err := repo.Config()
		if err != nil {
			return err
		}

		routingOption = cfg.Routing.Type
		if routingOption == "" {
			routingOption = routingOptionDHTKwd
		}
	}
	switch routingOption {
	case routingOptionSupernodeKwd:
		return errors.New("supernode routing was never fully implemented and has been removed")
	case routingOptionDHTClientKwd:
		ncfg.Routing = core.DHTClientOption
	case routingOptionDHTKwd:
		ncfg.Routing = core.DHTOption
	case routingOptionNoneKwd:
		ncfg.Routing = core.NilRouterOption
	default:
		return fmt.Errorf("unrecognized routing option: %s", routingOption)
	}

	node, err := core.NewNode(req.Context, ncfg)
	if err != nil {
		log.Error("error from node construction: ", err)
		return err
	}
	node.SetLocal(false)

	if node.PNetFingerprint != nil {
		fmt.Println("Swarm is limited to private network of peers with the swarm key")
		fmt.Printf("Swarm key fingerprint: %x\n", node.PNetFingerprint)
	}

	printSwarmAddrs(node)

	defer func() {
		// We wait for the node to close first, as the node has children
		// that it will wait for before closing, such as the API server.
		node.Close()

		select {
		case <-req.Context.Done():
			log.Info("Gracefully shut down daemon")
		default:
		}
	}()

	cctx.ConstructNode = func() (*core.IpfsNode, error) {
		return node, nil
	}

	// construct api endpoint - every time
	apiErrc, err := serveHTTPApi(req, cctx)
	if err != nil {
		return err
	}

	// construct fuse mountpoints - if the user provided the --mount flag
	mount, _ := req.Options[mountKwd].(bool)
	if mount && offline {
		return cmdkit.Errorf(cmdkit.ErrClient, "mount is not currently supported in offline mode")
	}
	if mount {
		if err := mountFuse(req, cctx); err != nil {
			return err
		}
	}

	// repo blockstore GC - if --enable-gc flag is present
	gcErrc, err := maybeRunGC(req, node)
	if err != nil {
		return err
	}

	// construct http gateway - if it is set in the config
	var gwErrc <-chan error
	if len(cfg.Addresses.Gateway) > 0 {
		var err error
		gwErrc, err = serveHTTPGateway(req, cctx)
		if err != nil {
			return err
		}
	}

	// initialize metrics collector
	prometheus.MustRegister(&corehttp.IpfsNodeCollector{Node: node})

	if !offline {
		commands.SetupBackupHandler(env)
		fmt.Println("backup function started")

		err = commands.RunBlacklistRefreshService(req.Context, env)
		if err != nil {
			return err
		}
		fmt.Println("run blacklist refresh service success")

		if cfg.Report.Address != "" {
			go reportWorker(node, req.Context)
		}
	}

	fmt.Printf("Daemon is ready\n")
	// collect long-running errors and block for shutdown
	// TODO(cryptix): our fuse currently doesnt follow this pattern for graceful shutdown
	for err := range merge(apiErrc, gwErrc, gcErrc) {
		if err != nil {
			return err
		}
	}

	return nil
}

// serveHTTPApi collects options, creates listener, prints status message and starts serving requests
func serveHTTPApi(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: GetConfig() failed: %s", err)
	}

	apiAddrs := make([]string, 0, 2)
	apiAddr, _ := req.Options[commands.ApiOption].(string)
	if apiAddr == "" {
		apiAddrs = cfg.Addresses.API
	} else {
		apiAddrs = append(apiAddrs, apiAddr)
	}

	listeners := make([]manet.Listener, 0, len(apiAddrs))
	for _, addr := range apiAddrs {
		apiMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: invalid API address: %q (err: %s)", apiAddr, err)
		}

		apiLis, err := manet.Listen(apiMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: manet.Listen(%s) failed: %s", apiMaddr, err)
		}

		// we might have listened to /tcp/0 - lets see what we are listing on
		apiMaddr = apiLis.Multiaddr()
		fmt.Printf("API server listening on %s\n", apiMaddr)

		listeners = append(listeners, apiLis)
	}

	// by default, we don't let you load arbitrary udfs objects through the api,
	// because this would open up the api to scripting vulnerabilities.
	// only the webui objects are allowed.
	// if you know what you're doing, go ahead and pass --unrestricted-api.
	unrestricted, _ := req.Options[unrestrictedApiAccessKwd].(bool)
	gatewayOpt := corehttp.GatewayOption(false, corehttp.WebUIPaths...)
	if unrestricted {
		gatewayOpt = corehttp.GatewayOption(true, "/ipfs", "/ipns")
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("api"),
		corehttp.CheckVersionOption(),
		corehttp.CommandsOption(*cctx),
		corehttp.WebUIOption,
		gatewayOpt,
		corehttp.VersionOption(),
		defaultMux("/debug/vars"),
		defaultMux("/debug/pprof/"),
		corehttp.MutexFractionOption("/debug/pprof-mutex/"),
		corehttp.MetricsScrapingOption("/debug/metrics/prometheus"),
		corehttp.LogOption(),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: ConstructNode() failed: %s", err)
	}

	if err := node.Repo.SetAPIAddr(listeners[0].Multiaddr()); err != nil {
		return nil, fmt.Errorf("serveHTTPApi: SetAPIAddr() failed: %s", err)
	}

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, apiLis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(apiLis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

// printSwarmAddrs prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	if !node.OnlineMode() {
		fmt.Println("Swarm not listening, running in offline mode.")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := node.PeerHost.Network().InterfaceListenAddresses()
	if err != nil {
		log.Errorf("failed to read listening addresses: %s", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Sort(sort.StringSlice(lisAddrs))
	for _, addr := range lisAddrs {
		fmt.Printf("Swarm listening on %s\n", addr)
	}

	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))
	for _, addr := range addrs {
		fmt.Printf("Swarm announcing %s\n", addr)
	}

}

// serveHTTPGateway collects options, creates listener, prints status message and starts serving requests
func serveHTTPGateway(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: GetConfig() failed: %s", err)
	}

	writable, writableOptionFound := req.Options[writableKwd].(bool)
	if !writableOptionFound {
		writable = cfg.Gateway.Writable
	}

	gatewayAddrs := cfg.Addresses.Gateway
	listeners := make([]manet.Listener, 0, len(gatewayAddrs))
	for _, addr := range gatewayAddrs {
		gatewayMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: invalid gateway address: %q (err: %s)", addr, err)
		}

		gwLis, err := manet.Listen(gatewayMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
		}
		// we might have listened to /tcp/0 - lets see what we are listing on
		gatewayMaddr = gwLis.Multiaddr()

		if writable {
			fmt.Printf("Gateway (writable) server listening on %s\n", gatewayMaddr)
		} else {
			fmt.Printf("Gateway (readonly) server listening on %s\n", gatewayMaddr)
		}

		listeners = append(listeners, gwLis)
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(writable, "/ipfs", "/ipns"),
		corehttp.VersionOption(),
		corehttp.CheckVersionOption(),
		corehttp.CommandsROOption(*cctx),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: ConstructNode() failed: %s", err)
	}

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, lis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(lis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

//collects options and opens the fuse mountpoint
func mountFuse(req *cmds.Request, cctx *oldcmds.Context) error {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return fmt.Errorf("mountFuse: GetConfig() failed: %s", err)
	}

	fsdir, found := req.Options[ipfsMountKwd].(string)
	if !found {
		fsdir = cfg.Mounts.IPFS
	}

	nsdir, found := req.Options[ipnsMountKwd].(string)
	if !found {
		nsdir = cfg.Mounts.IPNS
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return fmt.Errorf("mountFuse: ConstructNode() failed: %s", err)
	}

	err = nodeMount.Mount(node, fsdir, nsdir)
	if err != nil {
		return err
	}
	fmt.Printf("IPFS mounted at: %s\n", fsdir)
	fmt.Printf("IPNS mounted at: %s\n", nsdir)
	return nil
}

func maybeRunGC(req *cmds.Request, node *core.IpfsNode) (<-chan error, error) {
	enableGC, _ := req.Options[enableGCKwd].(bool)
	if !enableGC {
		return nil, nil
	}

	errc := make(chan error)
	go func() {
		errc <- corerepo.PeriodicGC(req.Context, node)
		close(errc)
	}()
	return errc, nil
}

// merge does fan-in of multiple read-only error channels
// taken from http://blog.golang.org/pipelines
func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	for _, c := range cs {
		if c != nil {
			wg.Add(1)
			go output(c)
		}
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func YesNoPrompt(prompt string) bool {
	var s string
	for i := 0; i < 3; i++ {
		fmt.Printf("%s ", prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		case "":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}

	return false
}


func printVersion() {
	fmt.Printf("go-ipfs version: %s-%s\n", version.CurrentVersionNumber, version.CurrentCommit)
	fmt.Printf("Repo version: %d\n", fsrepo.RepoVersion)
	fmt.Printf("System version: %s\n", runtime.GOARCH+"/"+runtime.GOOS)
	fmt.Printf("Golang version: %s\n", runtime.Version())
}


// ============================================================== report

/*
{
    "sign": "",
    "txid": "",
    "pubkey": "",
    "voutid": 1,
    "licperiod": 1234,
    "licversion": 1,
    "data": {
        "sign": "fkldjslkfjlasfjldj",
        "id": "",
        "ts": 12345,
        "in": 12,
        "out": 34,
        "storage": 431234,
        "list": [
            {
                "id": "",
                "in": 2,
                "out": 1
            }
        ]
    }
}
*/
type dataMetaListObject struct {
	Src string `json:"nodeId"`
	ID  string `json:"desNodeId"`
	In  int32  `json:"inflow"`
	Out int32  `json:"outflow"`
}

type reportData struct {
	Sign    string                `json:"sign,omitempty"`
	Account string `json:"account"`
	ID      string                `json:"id"`
	Ts      int64                 `json:"ts"`
	In      int32                 `json:"in"`
	Out     int32                 `json:"out"`
	Storage int32                 `json:"storage"`
	List    []*dataMetaListObject `json:"list,omitempty"`
}

type reportRequestBody struct {
	Sign       string      `json:"sign"`
	Txid       string      `json:"txid"`
	Pubkey     string      `json:"pubkey"`
	Voutid     int32       `json:"voutid"`
	Licperiod  int64       `json:"licperiod"`
	Licversion int32       `json:"licversion"`
	Data       *reportData `json:"data"`
}

type reportResonseBody struct {
	ErrorCode string `json:"errorCode"`
	Success   bool   `json:"success"`
	ErrorMsg  string `json:"errorMsg"`
}

func reportWorker(node *core.IpfsNode, ctx context.Context) {
	repo := node.Repo
	cfg, err := repo.Config()
	if err != nil {
		log.Error("get repo config failed:", err.Error())
		return
	}

	_, err = url.Parse(cfg.Report.Address)
	if err != nil {
		log.Error("parse config.Report.Address as a URL failed: ", err.Error())
		return
	}

	pubkey, err := ca.PublicKeyFromPrivateAddr(cfg.Verify.Secret)
	if err != nil {
		log.Error("got verify public key failed: ", err)
		return
	}

	for cfg.Verify.License == "" {
		time.Sleep(5 * time.Second)
		continue
	}

	min := int(cfg.Report.DurationMin.Seconds())
	max := int(cfg.Report.DurationMax.Seconds())

	if max <= min {
		log.Errorf("report duration max value must more then min value: min=%d max=%d\n", min, max)
		return
	}

	rand.Seed(time.Now().UnixNano())
	reportDurationDiff := max - min

	dur := time.Duration(min+rand.Intn(reportDurationDiff)) * time.Second
	log.Debug("report duration = ", dur)
	tm := time.NewTimer(dur)
	defer tm.Stop()

	cli := &http.Client{
		Timeout: cfg.Report.RequestTimeout.Duration,
	}

	rrb := reportRequestBody{
		Sign:       cfg.Verify.License,
		Txid:       cfg.Verify.Txid,
		Voutid:     cfg.Verify.Voutid,
		Pubkey:     pubkey,
		Licperiod:  cfg.Verify.Period,
		Licversion: cfg.Verify.Licversion,
	}

	bs, ok := node.Exchange.(*bitswap.Bitswap)
	if !ok {
		log.Error("exchange is not a bitswap object!")
		return
	}

	for {
		select {
		case <-tm.C:
			dur := time.Duration(min+rand.Intn(reportDurationDiff)) * time.Second
			log.Debug("report duration = ", dur)
			tm.Reset(dur)

			if cfg.Verify.License == "" {
				continue
			}

			data, err := buildReportData(node.Identity.Pretty(), bs, repo)
			if err != nil {
				log.Error("build report data failed:", err)
				continue
			}
			rrb.Data = data

			b, err := json.Marshal(rrb)
			if err != nil {
				log.Error("marshal report request body failed: ", err)
				continue
			}

			log.Debug("report request body: ", string(b))

			resp, err := cli.Post(cfg.Report.Address, "application/json", bytes.NewReader(b))
			if err != nil {
				log.Error("report error:", err.Error())
				continue
			}

			err = handleReportResponse(resp)
			if err != nil {
				log.Error("report failed:", err.Error())
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}

func buildReportData(src string, bs *bitswap.Bitswap, repo repo.Repo) (*reportData, error) {

	usage, err := repo.GetStorageUsage()
	if err != nil {
		return nil, errors.Wrap(err, "got repo storage usage error")
	}

	usage = usage / 1024 / 1024

	diffs := bs.AllLedgerAccountDiff()

	cfg, _ := repo.Config()

	data := &reportData{
		Account: cfg.Report.Account,
		ID:      cfg.Identity.PeerID,
		Storage: int32(usage),
		Ts:      time.Now().Unix(),
	}

	for _, diff := range diffs {
		data.List = append(data.List, &dataMetaListObject{
			Src: src,
			ID:  diff.ID,
			In:  int32(diff.RecvDiff),
			Out: int32(diff.SentDiff),
		})
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "marshal report data error")
	}

	uint256 := ca.NewSha2Hash(b)
	sign, err := ca.Sign(uint256.String(), cfg.Verify.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "sign report data error")
	}
	data.Sign = sign

	return data, nil

}

func handleReportResponse(resp *http.Response) error {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("report response code not 200: %d", resp.StatusCode)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read report response body error")
	}

	rrb := &reportResonseBody{}
	err = json.Unmarshal(bs, rrb)
	if err != nil {
		return errors.Wrapf(err, "unmarshal report response [body=%s]error", string(bs))
	}

	if rrb.ErrorCode != "OK" {
		return errors.New(string(bs))
	}
	return nil
}


func doVerify(repo repo.Repo, rcfg *config.Config, ) error {

	vfi := &rcfg.Verify
	if time.Now().After(time.Unix(vfi.Period, 0)) || vfi.License == "" || vfi.Licversion == 0 {
		log.Debug("request license...")

		lbi, err := ca.RequestLicense(rcfg.UCenter.ServerAddress, vfi.Txid, vfi.Voutid)
		if err != nil {
			return errors.Wrap(err, "request license failed")
		}
		vfi.License = lbi.License
		vfi.Period = lbi.LicPeriod
		vfi.Licversion = lbi.Licversion

		err = repo.SetConfig(rcfg)
		if err != nil {
			return errors.Wrap(err, "Save verify info to config file failed")
		}
	}

	// got pubkey
	pubkeyStr, err := ca.PublicKeyFromPrivateAddr(vfi.Secret)
	if err != nil {
		return errors.Wrap(err, "got pubkey failed")
	}

	// got server pubkey
	serverPubKey := ""
	for _, sp := range rcfg.UCenter.ServerPubkeys {
		if sp.Licversion == vfi.Licversion {
			serverPubKey = sp.Pubkey
			break
		}
	}
	if serverPubKey == "" {
		// request
		upm, err := ca.RequestUcenterPublicKeyMap(rcfg.UCenter.ServerAddress, vfi.Txid, vfi.Voutid)
		if err != nil {
			return errors.Wrap(err, "request ucenter public key map failed")
		}

		if len(upm.V2key) < len(rcfg.UCenter.ServerPubkeys) {
			return errors.New("request ucenter public key map less than already known")
		}

		var vps []*config.VersionPubkey
		for ver, key := range upm.V2key {
			vps = append(vps, &config.VersionPubkey{
				Licversion: ver,
				Pubkey:     key,
			})

			if ver == vfi.Licversion {
				serverPubKey = key
			}
		}
		rcfg.UCenter.ServerPubkeys = vps

		// save server public keys
		err = repo.SetConfig(rcfg)
		if err != nil {
			return errors.Wrap(err, "Save verify info to config file failed")
		}
	}

	// make node hash
	nodeHash := ca.MakeNodeInfoHash(vfi.Txid, int32(vfi.Voutid), pubkeyStr, vfi.Period, vfi.Licversion)

	// verify license
	ok, err := ca.VerifySignature(nodeHash, vfi.License, serverPubKey)
	if err != nil {
		return errors.Wrap(err, "verify license error")
	}

	if !ok {
		return errors.New("verify failed")
	}

	return nil
}