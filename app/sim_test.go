package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	"cosmossdk.io/x/feegrant"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	doapp "github.com/Daviddochain/dochain-core/v4/app"
	apptesting "github.com/Daviddochain/dochain-core/v4/app/testing"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simAppChainID = "simulation-app"

var emptySimulationWasmOpts []wasmkeeper.Option

func init() {
	simcli.GetSimulatorFlags()
}

func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

func fauxMerkleModeOpt(app *baseapp.BaseApp) {
	app.SetFauxMerkleMode()
}

func newSimulationApp(
	logger sdklog.Logger,
	db dbm.DB,
	homePath string,
	appOptions simtestutil.AppOptionsMap,
) *doapp.DoApp {
	return doapp.NewDoApp(
		logger,
		db,
		nil,
		true,
		map[int64]bool{},
		homePath,
		doapp.MakeEncodingConfig(),
		appOptions,
		emptySimulationWasmOpts,
		interBlockCacheOpt(),
		fauxMerkleModeOpt,
		baseapp.SetChainID(simAppChainID),
	)
}

func setupSimulationApp(
	t *testing.T,
	message string,
) (simtypes.Config, dbm.DB, simtestutil.AppOptionsMap, *doapp.DoApp) {
	t.Helper()
	if !apptesting.WasmVMAvailable {
		t.Skip("simulation tests require a CGO-enabled WasmVM build")
	}

	config := simcli.NewConfigFromFlags()
	config.ChainID = simAppChainID

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip(message)
	}
	require.NoError(t, err, "simulation setup failed")

	t.Cleanup(func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	})

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = dir
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue
	app := newSimulationApp(logger, db, dir, appOptions)
	require.Equal(t, "DoApp", app.Name())

	return config, db, appOptions, app
}

func runSimulation(
	t *testing.T,
	config simtypes.Config,
	app *doapp.DoApp,
) (bool, simulation.Params, error) {
	t.Helper()

	stopEarly, simParams, err := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		app.BaseApp,
		simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(app, app.AppCodec(), config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)

	return stopEarly, simParams, err
}

func TestFullAppSimulation(t *testing.T) {
	config, db, _, app := setupSimulationApp(t, "skipping full application simulation")

	_, simParams, simErr := runSimulation(t, config, app)
	require.NoError(t, simtestutil.CheckExportSimulation(app, config, simParams))
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}

func TestAppImportExport(t *testing.T) {
	config, db, appOptions, app := setupSimulationApp(t, "skipping import/export simulation")

	_, simParams, simErr := runSimulation(t, config, app)
	require.NoError(t, simtestutil.CheckExportSimulation(app, config, simParams))
	require.NoError(t, simErr)
	if config.Commit {
		simtestutil.PrintStats(db)
	}

	exported, err := app.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)

	newDB, newDir, logger, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-import",
		"Simulation-Import",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	})

	appOptions[flags.FlagHome] = newDir
	newApp := newSimulationApp(logger, newDB, newDir, appOptions)
	initReq := &abci.RequestInitChain{
		ChainId:       simAppChainID,
		AppStateBytes: exported.AppState,
	}

	ctxA := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	_, err = newApp.InitChainer(ctxB, initReq)
	if err != nil && strings.Contains(err.Error(), "validator set is empty after InitGenesis") {
		t.Logf("skipping import comparison because all validators were unbonded: %v\n%s", err, debug.Stack())
		return
	}
	require.NoError(t, err)
	require.NoError(t, newApp.StoreConsensusParams(ctxB, exported.ConsensusParams))

	skipPrefixes := map[string][][]byte{
		stakingtypes.StoreKey: {
			stakingtypes.UnbondingQueueKey,
			stakingtypes.RedelegationQueueKey,
			stakingtypes.ValidatorQueueKey,
			stakingtypes.HistoricalInfoKey,
			stakingtypes.UnbondingIDKey,
			stakingtypes.UnbondingIndexKey,
			stakingtypes.UnbondingTypeKey,
			stakingtypes.ValidatorUpdatesKey,
		},
		authzkeeper.StoreKey:   {authzkeeper.GrantQueuePrefix},
		feegrant.StoreKey:      {feegrant.FeeAllowanceQueueKeyPrefix},
		slashingtypes.StoreKey: {slashingtypes.ValidatorMissedBlockBitmapKeyPrefix},
		wasmtypes.StoreKey:     {wasmtypes.TXCounterPrefix},
	}

	storeKeys := app.GetKVStoreKey()
	require.NotEmpty(t, storeKeys)
	for keyName, appKeyA := range storeKeys {
		appKeyB := newApp.GetKey(keyName)
		require.NotNil(t, appKeyB, "imported app is missing store %q", keyName)

		failedKVAs, failedKVBs := simtestutil.DiffKVStores(
			ctxA.KVStore(appKeyA),
			ctxB.KVStore(appKeyB),
			skipPrefixes[keyName],
		)
		if !assert.Equal(t, len(failedKVAs), len(failedKVBs), "unequal key/value sets for store %q", keyName) {
			t.FailNow()
		}
		if !assert.Empty(
			t,
			failedKVAs,
			simtestutil.GetSimulationLog(keyName, app.SimulationManager().StoreDecoders, failedKVAs, failedKVBs),
		) {
			t.FailNow()
		}
	}
}

func TestAppSimulationAfterImport(t *testing.T) {
	config, db, appOptions, app := setupSimulationApp(t, "skipping simulation after import")

	stopEarly, simParams, simErr := runSimulation(t, config, app)
	require.NoError(t, simtestutil.CheckExportSimulation(app, config, simParams))
	require.NoError(t, simErr)
	if config.Commit {
		simtestutil.PrintStats(db)
	}
	if stopEarly {
		t.Skip("simulation stopped with no validator set to export")
	}

	exported, err := app.ExportAppStateAndValidators(true, []string{}, []string{})
	require.NoError(t, err)

	newDB, newDir, logger, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-after-import",
		"Simulation-After-Import",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	})

	appOptions[flags.FlagHome] = newDir
	newApp := newSimulationApp(logger, newDB, newDir, appOptions)
	_, err = newApp.InitChain(&abci.RequestInitChain{
		ChainId:       simAppChainID,
		AppStateBytes: exported.AppState,
	})
	require.NoError(t, err)

	_, _, err = runSimulation(t, config, newApp)
	require.NoError(t, err)
}

func TestAppStateDeterminism(t *testing.T) {
	if !apptesting.WasmVMAvailable {
		t.Skip("simulation tests require a CGO-enabled WasmVM build")
	}
	if !simcli.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := simcli.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = simAppChainID

	const attemptsPerSeed = 3
	numSeeds := 3
	if config.Seed != simcli.DefaultSeedValue {
		numSeeds = 1
	}
	baseSeed := config.Seed

	for seedIndex := 0; seedIndex < numSeeds; seedIndex++ {
		config.Seed = baseSeed + int64(seedIndex)
		var expectedHash json.RawMessage

		for attempt := 0; attempt < attemptsPerSeed; attempt++ {
			db := dbm.NewMemDB()
			homePath := t.TempDir()
			appOptions := make(simtestutil.AppOptionsMap)
			appOptions[flags.FlagHome] = homePath
			appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue
			app := newSimulationApp(sdklog.NewNopLogger(), db, homePath, appOptions)

			fmt.Printf(
				"running determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed,
				seedIndex+1,
				numSeeds,
				attempt+1,
				attemptsPerSeed,
			)

			_, _, err := runSimulation(t, config, app)
			require.NoError(t, err)
			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHash := json.RawMessage(app.LastCommitID().Hash)
			if attempt == 0 {
				expectedHash = append(json.RawMessage(nil), appHash...)
			} else {
				require.Equal(
					t,
					string(expectedHash),
					string(appHash),
					"non-determinism for seed %d on attempt %d",
					config.Seed,
					attempt+1,
				)
			}
			require.NoError(t, db.Close())
		}
	}
}
