package node

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bufbuild/protovalidate-go"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	bidderregistry "github.com/primev/mev-commit/contracts-abi/clients/BidderRegistry"
	blocktracker "github.com/primev/mev-commit/contracts-abi/clients/BlockTracker"
	preconf "github.com/primev/mev-commit/contracts-abi/clients/PreconfManager"
	providerregistry "github.com/primev/mev-commit/contracts-abi/clients/ProviderRegistry"
	validatorrouter "github.com/primev/mev-commit/contracts-abi/clients/ValidatorOptInRouter"
	bidderapiv1 "github.com/primev/mev-commit/p2p/gen/go/bidderapi/v1"
	debugapiv1 "github.com/primev/mev-commit/p2p/gen/go/debugapi/v1"
	notificationsapiv1 "github.com/primev/mev-commit/p2p/gen/go/notificationsapi/v1"
	preconfpb "github.com/primev/mev-commit/p2p/gen/go/preconfirmation/v1"
	providerapiv1 "github.com/primev/mev-commit/p2p/gen/go/providerapi/v1"
	validatorapiv1 "github.com/primev/mev-commit/p2p/gen/go/validatorapi/v1"
	"github.com/primev/mev-commit/p2p/pkg/apiserver"
	"github.com/primev/mev-commit/p2p/pkg/autodepositor"
	autodepositorstore "github.com/primev/mev-commit/p2p/pkg/autodepositor/store"
	"github.com/primev/mev-commit/p2p/pkg/crypto"
	"github.com/primev/mev-commit/p2p/pkg/depositmanager"
	depositmanagerstore "github.com/primev/mev-commit/p2p/pkg/depositmanager/store"
	"github.com/primev/mev-commit/p2p/pkg/discovery"
	"github.com/primev/mev-commit/p2p/pkg/keyexchange"
	"github.com/primev/mev-commit/p2p/pkg/keysstore"
	"github.com/primev/mev-commit/p2p/pkg/notifications"
	"github.com/primev/mev-commit/p2p/pkg/p2p"
	"github.com/primev/mev-commit/p2p/pkg/p2p/libp2p"
	"github.com/primev/mev-commit/p2p/pkg/preconfirmation"
	preconfencryptor "github.com/primev/mev-commit/p2p/pkg/preconfirmation/encryptor"
	preconfstore "github.com/primev/mev-commit/p2p/pkg/preconfirmation/store"
	preconftracker "github.com/primev/mev-commit/p2p/pkg/preconfirmation/tracker"
	bidderapi "github.com/primev/mev-commit/p2p/pkg/rpc/bidder"
	debugapi "github.com/primev/mev-commit/p2p/pkg/rpc/debug"
	notificationsapi "github.com/primev/mev-commit/p2p/pkg/rpc/notifications"
	providerapi "github.com/primev/mev-commit/p2p/pkg/rpc/provider"
	validatorapi "github.com/primev/mev-commit/p2p/pkg/rpc/validator"
	"github.com/primev/mev-commit/p2p/pkg/signer"
	"github.com/primev/mev-commit/p2p/pkg/storage"
	inmem "github.com/primev/mev-commit/p2p/pkg/storage/inmem"
	pebblestorage "github.com/primev/mev-commit/p2p/pkg/storage/pebble"
	"github.com/primev/mev-commit/p2p/pkg/topology"
	"github.com/primev/mev-commit/p2p/pkg/txnstore"
	"github.com/primev/mev-commit/x/contracts/events"
	"github.com/primev/mev-commit/x/contracts/events/publisher"
	"github.com/primev/mev-commit/x/contracts/transactor"
	"github.com/primev/mev-commit/x/contracts/txmonitor"
	"github.com/primev/mev-commit/x/health"
	"github.com/primev/mev-commit/x/keysigner"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcServerDialTimeout = 5 * time.Second
)

type Options struct {
	Version                  string
	DataDir                  string
	KeySigner                keysigner.KeySigner
	Secret                   string
	PeerType                 string
	Logger                   *slog.Logger
	P2PPort                  int
	P2PAddr                  string
	HTTPAddr                 string
	RPCAddr                  string
	Bootnodes                []string
	PreconfContract          string
	BlockTrackerContract     string
	ProviderRegistryContract string
	BidderRegistryContract   string
	ValidatorRouterContract  string
	AutodepositAmount        *big.Int
	RPCEndpoint              string
	WSRPCEndpoint            string
	NatAddr                  string
	TLSCertificateFile       string
	TLSPrivateKeyFile        string
	ProviderWhitelist        []common.Address
	DefaultGasLimit          uint64
	DefaultGasTipCap         *big.Int
	DefaultGasFeeCap         *big.Int
	OracleWindowOffset       *big.Int
	BeaconAPIURL             string
	L1RPCURL                 string
	LaggardMode              *big.Int
	BidderBidTimeout         time.Duration
	ProviderDecisionTimeout  time.Duration
	NotificationsBufferCap   int
}

type Node struct {
	cancelFunc context.CancelFunc
	closers    []io.Closer
}

func NewNode(opts *Options) (*Node, error) {
	nd := &Node{
		closers: make([]io.Closer, 0),
	}

	srv := apiserver.New(opts.Version, opts.Logger.With("component", "apiserver"))
	peerType := p2p.FromString(opts.PeerType)

	var (
		contractRPC *ethclient.Client
		err         error
	)
	if opts.WSRPCEndpoint != "" {
		contractRPC, err = ethclient.Dial(opts.WSRPCEndpoint)
		if err != nil {
			opts.Logger.Error("failed to connect to ws rpc", "error", err)
			return nil, err
		}
	} else {
		contractRPC, err = ethclient.Dial(opts.RPCEndpoint)
		if err != nil {
			opts.Logger.Error("failed to connect to rpc", "error", err)
			return nil, err
		}
	}

	progressstore := &progressStore{contractRPC: contractRPC}

	chainID, err := contractRPC.ChainID(context.Background())
	if err != nil {
		opts.Logger.Error("failed to get chain ID", "error", err)
		return nil, err
	}

	var store storage.Storage
	if opts.DataDir != "" {
		store, err = pebblestorage.New(opts.DataDir)
		if err != nil {
			opts.Logger.Error("failed to create storage", "error", err)
			return nil, err
		}
	} else {
		store = inmem.New()
	}
	nd.closers = append(nd.closers, store)

	contracts, err := getContractABIs(opts)
	if err != nil {
		opts.Logger.Error("failed to get contract ABIs", "error", err)
		return nil, err
	}

	abis := make([]*abi.ABI, 0, len(contracts))
	contractAddrs := make([]common.Address, 0, len(contracts))

	for addr, abi := range contracts {
		abis = append(abis, abi)
		contractAddrs = append(contractAddrs, addr)
	}

	evtMgr := events.NewListener(
		opts.Logger.With("component", "events"),
		abis...,
	)
	srv.RegisterMetricsCollectors(evtMgr.Metrics()...)

	var startables []StartableObjWithDesc
	var evtPublisher PublisherStartable

	if opts.WSRPCEndpoint != "" {
		// Use WS publisher if WSRPCEndpoint is set
		evtPublisher = publisher.NewWSPublisher(
			progressstore,
			opts.Logger.With("component", "ws_publisher"),
			contractRPC,
			evtMgr,
		)
	} else {
		evtPublisher = publisher.NewHTTPPublisher(
			progressstore,
			opts.Logger.With("component", "http_publisher"),
			contractRPC,
			evtMgr,
		)
	}

	startables = append(
		startables,
		StartableObjWithDesc{
			Desc: "events_publisher",
			Startable: StartableFunc(func(ctx context.Context) <-chan struct{} {
				return evtPublisher.Start(ctx, contractAddrs...)
			}),
		},
	)

	txnStore := txnstore.New(store)

	monitor := txmonitor.New(
		opts.KeySigner.GetAddress(),
		contractRPC,
		txmonitor.NewEVMHelperWithLogger(contractRPC, opts.Logger.With("component", "txmonitor"), contracts),
		txnStore,
		opts.Logger.With("component", "txmonitor"),
		1024,
	)
	startables = append(
		startables,
		StartableObjWithDesc{
			Desc:      "txmonitor",
			Startable: monitor,
		},
	)
	srv.RegisterMetricsCollectors(monitor.Metrics()...)

	contractsBackend := transactor.NewMetricsWrapper(
		transactor.NewTransactor(
			contractRPC,
			monitor,
		),
	)
	srv.RegisterMetricsCollectors(contractsBackend.Metrics()...)

	providerRegistry, err := providerregistry.NewProviderregistry(
		common.HexToAddress(opts.ProviderRegistryContract),
		contractsBackend,
	)
	if err != nil {
		opts.Logger.Error("failed to instantiate provider registry contract", "error", err)
		return nil, err
	}

	bidderRegistry, err := bidderregistry.NewBidderregistry(
		common.HexToAddress(opts.BidderRegistryContract),
		contractsBackend,
	)
	if err != nil {
		opts.Logger.Error("failed to instantiate bidder registry contract", "error", err)
		return nil, err
	}

	optsGetter := func(ctx context.Context) (*bind.TransactOpts, error) {
		tOpts, err := opts.KeySigner.GetAuthWithCtx(ctx, chainID)
		if err == nil {
			// Use any defaults set by user
			tOpts.GasLimit = opts.DefaultGasLimit
			tOpts.GasTipCap = opts.DefaultGasTipCap
			tOpts.GasFeeCap = opts.DefaultGasFeeCap
		}
		return tOpts, err
	}

	keysStore := keysstore.New(store)

	p2pSvc, err := libp2p.New(&libp2p.Options{
		KeySigner: opts.KeySigner,
		Secret:    opts.Secret,
		PeerType:  peerType,
		Register: &providerStakeChecker{
			providerRegistry: providerRegistry,
			from:             opts.KeySigner.GetAddress(),
		},
		Store:          keysStore,
		Logger:         opts.Logger.With("component", "p2p"),
		ListenPort:     opts.P2PPort,
		ListenAddr:     opts.P2PAddr,
		MetricsReg:     srv.MetricsRegistry(),
		BootstrapAddrs: opts.Bootnodes,
		NatAddr:        opts.NatAddr,
	})
	if err != nil {
		opts.Logger.Error("failed to create p2p service", "error", err)
		return nil, err
	}
	nd.closers = append(nd.closers, p2pSvc)

	notificationsSvc := notifications.New(opts.NotificationsBufferCap)
	nd.closers = append(
		nd.closers,
		ioCloserFunc(func() error {
			notificationsSvc.Shutdown()
			return nil
		}),
	)

	topo := topology.New(p2pSvc, notificationsSvc, opts.Logger.With("component", "topology"))
	disc := discovery.New(topo, p2pSvc, opts.Logger.With("component", "discovery_protocol"))
	nd.closers = append(nd.closers, disc)

	srv.RegisterMetricsCollectors(topo.Metrics()...)

	// Set the announcer for the topology service
	topo.SetAnnouncer(disc)
	// Set the notifier for the p2p service
	p2pSvc.SetNotifier(topo)

	// Register the discovery protocol with the p2p service
	p2pSvc.AddStreamHandlers(disc.Streams()...)

	lis, err := net.Listen("tcp", opts.RPCAddr)
	if err != nil {
		opts.Logger.Error("failed to listen", "error", err)
		return nil, errors.Join(err, nd.Close())
	}

	var tlsCredentials credentials.TransportCredentials
	if opts.TLSCertificateFile != "" && opts.TLSPrivateKeyFile != "" {
		tlsCredentials, err = credentials.NewServerTLSFromFile(
			opts.TLSCertificateFile,
			opts.TLSPrivateKeyFile,
		)
		if err != nil {
			opts.Logger.Error("failed to load TLS credentials", "error", err)
			return nil, fmt.Errorf("unable to load TLS credentials: %w", err)
		}
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(tlsCredentials),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	debugService := debugapi.NewService(
		txnStore,
		txmonitor.NewCanceller(
			chainID,
			contractRPC,
			opts.KeySigner,
			monitor,
			opts.Logger.With("component", "txmonitor/canceller"),
		),
		p2pSvc,
		topo,
	)

	debugapiv1.RegisterDebugServiceServer(grpcServer, debugService)

	notificationsRPCService := notificationsapi.NewService(
		notificationsSvc,
		opts.Logger.With("component", "notifications"),
	)

	notificationsapiv1.RegisterNotificationsServer(grpcServer, notificationsRPCService)

	var autoDeposit *autodepositor.AutoDepositTracker

	if opts.PeerType != p2p.PeerTypeBootnode.String() {
		validator, err := protovalidate.New()
		if err != nil {
			opts.Logger.Error("failed to create proto validator", "error", err)
			return nil, errors.Join(err, nd.Close())
		}

		var (
			bidProcessor preconfirmation.BidProcessor   = noOpBidProcessor{}
			depositMgr   preconfirmation.DepositManager = noOpDepositManager{}
		)

		blockTrackerCaller, err := blocktracker.NewBlocktrackerCaller(
			common.HexToAddress(opts.BlockTrackerContract),
			contractRPC,
		)
		if err != nil {
			opts.Logger.Error("failed to instantiate block tracker contract", "error", err)
			return nil, err
		}

		blockTrackerSession := &blocktracker.BlocktrackerCallerSession{
			Contract: blockTrackerCaller,
			CallOpts: bind.CallOpts{
				From: opts.KeySigner.GetAddress(),
			},
		}

		commitmentDA, err := preconf.NewPreconfmanager(
			common.HexToAddress(opts.PreconfContract),
			contractsBackend,
		)
		if err != nil {
			opts.Logger.Error("failed to instantiate preconf commitment store contract", "error", err)
			return nil, err
		}

		var (
			pk *bn254.G1Affine
			sk *fr.Element
		)
		if peerType == p2p.PeerTypeProvider {
			pk, err = keysStore.BN254PublicKey()
			if err != nil {
				opts.Logger.Error("failed to get bn254 public key", "error", err)
				return nil, err
			}
			sk, err = keysStore.BN254PrivateKey()
			if err != nil {
				opts.Logger.Error("failed to get bn254 secret key", "error", err)
				return nil, err
			}
		}
		tracker := preconftracker.NewTracker(
			chainID,
			peerType,
			opts.KeySigner.GetAddress(),
			evtMgr,
			preconfstore.New(store),
			commitmentDA,
			txmonitor.NewEVMHelperWithLogger(contractRPC, opts.Logger.With("component", "evm_helper"), contracts),
			pk,
			sk,
			optsGetter,
			opts.Logger.With("component", "tracker"),
		)
		startables = append(
			startables,
			StartableObjWithDesc{
				Desc:      "tracker",
				Startable: tracker,
			},
		)
		srv.RegisterMetricsCollectors(tracker.Metrics()...)

		bpwBigInt, err := blockTrackerSession.GetBlocksPerWindow()
		if err != nil {
			opts.Logger.Error("failed to get blocks per window", "error", err)
			return nil, err
		}

		l1ContractRPC, err := ethclient.Dial(opts.L1RPCURL)
		if err != nil {
			opts.Logger.Error("failed to connect to rpc", "error", err)
			return nil, err
		}

		validatorRouterCaller, err := validatorrouter.NewValidatoroptinrouterCaller(
			common.HexToAddress(opts.ValidatorRouterContract),
			l1ContractRPC,
		)
		if err != nil {
			opts.Logger.Error("failed to instantiate validator router contract", "error", err)
			return nil, err
		}

		callOptsGetter := func() (*bind.CallOpts, error) {
			blkNum, err := l1ContractRPC.BlockNumber(context.Background())
			if err != nil {
				return nil, err
			}
			currentBlkNum := big.NewInt(0).SetUint64(blkNum)
			queryBlkNum := big.NewInt(0).Sub(currentBlkNum, opts.LaggardMode)
			return &bind.CallOpts{
				From:        opts.KeySigner.GetAddress(),
				BlockNumber: queryBlkNum,
			}, nil
		}

		validatorAPI := validatorapi.NewService(
			opts.BeaconAPIURL,
			validatorRouterCaller,
			opts.Logger.With("component", "validatorapi"),
			callOptsGetter,
			notificationsSvc,
		)
		if err != nil {
			opts.Logger.Error("failed to create validator api", "error", err)
			return nil, err
		}
		validatorapiv1.RegisterValidatorServer(grpcServer, validatorAPI)
		startables = append(
			startables,
			StartableObjWithDesc{
				Desc:      "validators",
				Startable: validatorAPI,
			},
		)
		blocksPerWindow := bpwBigInt.Uint64()

		switch opts.PeerType {
		case p2p.PeerTypeProvider.String():
			providerAPI := providerapi.NewService(
				opts.Logger.With("component", "providerapi"),
				providerRegistry,
				opts.KeySigner.GetAddress(),
				monitor,
				optsGetter,
				validator,
			)
			providerapiv1.RegisterProviderServer(grpcServer, providerAPI)
			bidProcessor = providerAPI
			srv.RegisterMetricsCollectors(providerAPI.Metrics()...)
			depositMgr = depositmanager.NewDepositManager(
				blocksPerWindow,
				depositmanagerstore.New(store),
				evtMgr,
				bidderRegistry,
				opts.Logger.With("component", "depositmanager"),
			)
			startables = append(
				startables,
				StartableObjWithDesc{
					Desc:      "deposit_manager",
					Startable: depositMgr.(*depositmanager.DepositManager),
				},
			)
			preconfEncryptor, err := preconfencryptor.NewEncryptor(opts.KeySigner, keysStore, chainID, opts.PreconfContract)
			if err != nil {
				opts.Logger.Error("failed to create preconf encryptor", "error", err)
				return nil, errors.Join(err, nd.Close())
			}
			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfEncryptor,
				depositMgr,
				bidProcessor,
				commitmentDA,
				tracker,
				optsGetter,
				opts.ProviderDecisionTimeout,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			// Only register handler for provider
			p2pSvc.AddStreamHandlers(preconfProto.Streams()...)
			keyexchange := keyexchange.New(
				topo,
				p2pSvc,
				opts.KeySigner,
				nil,
				keysStore,
				opts.Logger.With("component", "keyexchange_protocol"),
				signer.New(),
				nil,
			)
			p2pSvc.AddStreamHandlers(keyexchange.Streams()...)
			srv.RegisterMetricsCollectors(preconfProto.Metrics()...)

		case p2p.PeerTypeBidder.String():
			aesKey, err := crypto.GenerateAESKey()
			if err != nil {
				opts.Logger.Error("failed to generate AES key", "error", err)
				return nil, errors.Join(err, nd.Close())
			}
			err = keysStore.SetAESKey(opts.KeySigner.GetAddress(), aesKey)
			if err != nil {
				opts.Logger.Error("failed to set AES key", "error", err)
				return nil, errors.Join(err, nd.Close())
			}

			preconfEncryptor, err := preconfencryptor.NewEncryptor(opts.KeySigner, keysStore, chainID, opts.PreconfContract)
			if err != nil {
				opts.Logger.Error("failed to create preconf encryptor", "error", err)
				return nil, errors.Join(err, nd.Close())
			}

			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfEncryptor,
				depositMgr,
				bidProcessor,
				commitmentDA,
				tracker,
				optsGetter,
				opts.ProviderDecisionTimeout,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			srv.RegisterMetricsCollectors(preconfProto.Metrics()...)

			autodepositorStore := autodepositorstore.New(store)

			autoDeposit = autodepositor.New(
				evtMgr,
				bidderRegistry,
				blockTrackerSession,
				optsGetter,
				autodepositorStore,
				opts.OracleWindowOffset,
				opts.Logger.With("component", "auto_deposit_tracker"),
			)

			nd.closers = append(
				nd.closers,
				ioCloserFunc(func() error {
					_, err := autoDeposit.Stop()
					return err
				}),
			)

			bidderAPI := bidderapi.NewService(
				opts.KeySigner.GetAddress(),
				blocksPerWindow,
				preconfProto,
				bidderRegistry,
				blockTrackerSession,
				validator,
				monitor,
				optsGetter,
				autoDeposit,
				autodepositorStore,
				opts.OracleWindowOffset,
				opts.BidderBidTimeout,
				opts.Logger.With("component", "bidderapi"),
			)
			bidderapiv1.RegisterBidderServer(grpcServer, bidderAPI)

			keyexchange := keyexchange.New(
				topo,
				p2pSvc,
				opts.KeySigner,
				aesKey,
				keysStore,
				opts.Logger.With("component", "keyexchange_protocol"),
				signer.New(),
				opts.ProviderWhitelist,
			)
			go func() {
				sub := notificationsSvc.Subscribe(notifications.TopicPeerConnected)
				for p := range sub {
					peerType, ok := p.Value()["type"].(string)
					if ok && peerType == p2p.PeerTypeProvider.String() {
						err = keyexchange.SendTimestampMessage()
						if err != nil {
							opts.Logger.Error("failed to send timestamp message", "error", err)
						}
					}
				}
			}()

			srv.RegisterMetricsCollectors(bidderAPI.Metrics()...)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	nd.cancelFunc = cancel
	healthChecker := health.New()

	for _, s := range startables {
		closeChan := s.Startable.Start(ctx)
		healthChecker.Register(health.CloseChannelHealthCheck(s.Desc, closeChan))
		nd.closers = append(nd.closers, channelCloserFunc(closeChan))
	}

	if opts.AutodepositAmount != nil && autoDeposit != nil {
		err = autoDeposit.Start(ctx, nil, opts.AutodepositAmount)
		if err != nil {
			opts.Logger.Error("failed to start auto deposit tracker", "error", err)
			return nil, errors.Join(err, nd.Close())
		}
	}

	started := make(chan struct{})
	go func() {
		// signal that the server has started
		close(started)

		err := grpcServer.Serve(lis)
		if err != nil {
			opts.Logger.Error("failed to start grpc server", "err", err)
		}
	}()
	nd.closers = append(nd.closers, lis)

	// Wait for the server to start
	<-started

	// Since we don't know if the server has TLS enabled on its rpc
	// endpoint, we try different strategies from most secure to
	// least secure. In the future, when only TLS-enabled servers
	// are allowed, only the TLS system pool certificate strategy
	// should be used.
	var grpcConn *grpc.ClientConn
	for _, e := range []struct {
		strategy   string
		isSecure   bool
		credential credentials.TransportCredentials
	}{
		{"TLS system pool certificate", true, credentials.NewClientTLSFromCert(nil, "")},
		{"TLS skip verification", false, credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})},
		{"TLS disabled", false, insecure.NewCredentials()},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), grpcServerDialTimeout)
		opts.Logger.Info("dialing to grpc server", "strategy", e.strategy)
		// nolint:staticcheck
		grpcConn, err = grpc.DialContext(
			ctx,
			opts.RPCAddr,
			grpc.WithBlock(),
			grpc.WithTransportCredentials(e.credential),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
		if err != nil {
			opts.Logger.Warn("failed to dial grpc server", "error", err)
			cancel()
			continue
		}

		cancel()
		if !e.isSecure {
			opts.Logger.Warn("established connection with the grpc server has potential security risk")
		}
		break
	}
	if grpcConn == nil {
		return nil, errors.New("dialing of grpc server failed")
	}

	healthChecker.Register(health.GrpcGatewayHealthCheck(grpcConn))

	handlerCtx, handlerCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer handlerCancel()

	gatewayMux := runtime.NewServeMux()
	err = debugapiv1.RegisterDebugServiceHandler(handlerCtx, gatewayMux, grpcConn)
	if err != nil {
		opts.Logger.Error("failed to register debug handler", "err", err)
		return nil, errors.Join(err, nd.Close())
	}

	err = notificationsapiv1.RegisterNotificationsHandler(handlerCtx, gatewayMux, grpcConn)
	if err != nil {
		opts.Logger.Error("failed to register notifications handler", "err", err)
		return nil, errors.Join(err, nd.Close())
	}

	err = validatorapiv1.RegisterValidatorHandler(handlerCtx, gatewayMux, grpcConn)
	if err != nil {
		opts.Logger.Error("failed to register validator handler", "err", err)
		return nil, errors.Join(err, nd.Close())
	}

	switch opts.PeerType {
	case p2p.PeerTypeProvider.String():
		err := providerapiv1.RegisterProviderHandler(handlerCtx, gatewayMux, grpcConn)
		if err != nil {
			opts.Logger.Error("failed to register provider handler", "err", err)
			return nil, errors.Join(err, nd.Close())
		}
	case p2p.PeerTypeBidder.String():
		err := bidderapiv1.RegisterBidderHandler(handlerCtx, gatewayMux, grpcConn)
		if err != nil {
			opts.Logger.Error("failed to register bidder handler", "err", err)
			return nil, errors.Join(err, nd.Close())
		}
	}

	srv.ChainHandlers("/", gatewayMux)
	srv.ChainHandlers(
		"/health",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if err := healthChecker.Health(); err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "ok")
			},
		),
	)

	server := &http.Server{
		Addr:    opts.HTTPAddr,
		Handler: srv.Router(),
	}

	go func() {
		var (
			err        error
			tlsEnabled = opts.TLSCertificateFile != "" && opts.TLSPrivateKeyFile != ""
		)
		opts.Logger.Info("starting to listen", "tls", tlsEnabled)
		if tlsEnabled {
			err = server.ListenAndServeTLS(
				opts.TLSCertificateFile,
				opts.TLSPrivateKeyFile,
			)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			opts.Logger.Error("failed to start server", "err", err)
		}
	}()
	nd.closers = append(nd.closers, server)

	return nd, nil
}

func getContractABIs(opts *Options) (map[common.Address]*abi.ABI, error) {
	abis := make(map[common.Address]*abi.ABI)

	btABI, err := abi.JSON(strings.NewReader(blocktracker.BlocktrackerABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.BlockTrackerContract)] = &btABI

	pcABI, err := abi.JSON(strings.NewReader(preconf.PreconfmanagerABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.PreconfContract)] = &pcABI

	brABI, err := abi.JSON(strings.NewReader(bidderregistry.BidderregistryABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.BidderRegistryContract)] = &brABI

	vrABI, err := abi.JSON(strings.NewReader(validatorrouter.ValidatoroptinrouterABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.ValidatorRouterContract)] = &vrABI

	return abis, nil
}

func (n *Node) Close() error {
	if n.cancelFunc != nil {
		n.cancelFunc()
	}

	var err error
	for _, c := range n.closers {
		err = errors.Join(err, c.Close())
	}

	return err
}

type noOpBidProcessor struct{}

// ProcessBid auto accepts all bids sent.
func (noOpBidProcessor) ProcessBid(
	_ context.Context,
	_ *preconfpb.Bid,
) (chan providerapi.ProcessedBidResponse, error) {
	statusC := make(chan providerapi.ProcessedBidResponse, 5)
	statusC <- providerapi.ProcessedBidResponse{Status: providerapiv1.BidResponse_STATUS_ACCEPTED, DispatchTimestamp: time.Now().UnixMilli()}
	close(statusC)

	return statusC, nil
}

type noOpDepositManager struct{}

func (noOpDepositManager) CheckAndDeductDeposit(_ context.Context, _ common.Address, _ string, _ int64) (func() error, error) {
	return func() error { return nil }, nil
}

type channelCloser <-chan struct{}

func channelCloserFunc(c <-chan struct{}) io.Closer {
	return channelCloser(c)
}

func (c channelCloser) Close() error {
	select {
	case <-c:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timeout waiting for channel to close")
	}
}

type ioCloserFunc func() error

func (f ioCloserFunc) Close() error {
	return f()
}

type PublisherStartable interface {
	Start(ctx context.Context, contracts ...common.Address) <-chan struct{}
}

type StartableObjWithDesc struct {
	Startable Startable
	Desc      string
}

type Startable interface {
	Start(ctx context.Context) <-chan struct{}
}

type StartableFunc func(ctx context.Context) <-chan struct{}

func (f StartableFunc) Start(ctx context.Context) <-chan struct{} {
	return f(ctx)
}

type providerStakeChecker struct {
	providerRegistry *providerregistry.Providerregistry
	from             common.Address
}

func (p *providerStakeChecker) CheckProviderRegistered(ctx context.Context, provider common.Address) bool {
	callOpts := &bind.CallOpts{
		From:    p.from,
		Context: ctx,
	}

	minStake, err := p.providerRegistry.MinStake(callOpts)
	if err != nil {
		return false
	}

	stake, err := p.providerRegistry.GetProviderStake(callOpts, provider)
	if err != nil {
		return false
	}

	return stake.Cmp(minStake) >= 0
}

type progressStore struct {
	contractRPC *ethclient.Client
	lastBlock   atomic.Uint64
}

func (p *progressStore) LastBlock() (uint64, error) {
	return p.contractRPC.BlockNumber(context.Background())
}

func (p *progressStore) SetLastBlock(block uint64) error {
	p.lastBlock.Store(block)
	return nil
}
