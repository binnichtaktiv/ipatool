package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/binnichtaktiv/ipatool/pkg/keychain"
	cookiejar "github.com/juju/persistent-cookiejar"
	"github.com/binnichtaktiv/ipatool/pkg/appstore"
	"github.com/binnichtaktiv/ipatool/pkg/http"
	"github.com/binnichtaktiv/ipatool/pkg/log"
	"github.com/binnichtaktiv/ipatool/pkg/util"
	"github.com/binnichtaktiv/ipatool/pkg/util/machine"
	"github.com/binnichtaktiv/ipatool/pkg/util/operatingsystem"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var dependencies = Dependencies{}
var keychainPassphrase string

type Dependencies struct {
	Logger    log.Logger
	OS        operatingsystem.OperatingSystem
	Machine   machine.Machine
	CookieJar http.CookieJar
	Keychain  keychain.Keychain
	AppStore  appstore.AppStore
}

// newLogger returns a new logger instance.
func newLogger(format OutputFormat, verbose bool) log.Logger {
	var writer io.Writer

	switch format {
	case OutputFormatJSON:
		writer = zerolog.SyncWriter(os.Stdout)
	case OutputFormatText:
		writer = log.NewWriter()
	}

	return log.NewLogger(log.Args{
		Verbose: verbose,
		Writer:  writer,
	},
	)
}

// newCookieJar returns a new cookie jar instance.
func newCookieJar(machine machine.Machine) http.CookieJar {
	return util.Must(cookiejar.New(&cookiejar.Options{
		Filename: filepath.Join(machine.HomeDirectory(), ConfigDirectoryName, CookieJarFileName),
	}))
}

// newKeychain returns a new keychain instance.
func newKeychain(machine machine.Machine, logger log.Logger, interactive bool) keychain.Keychain {
	// Use home directory for keychain storage to avoid permission issues
	keyringFilePath := filepath.Join(machine.HomeDirectory(), ".ipatool", "ipatool-auth.json")
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(keyringFilePath), 0700); err != nil {
		logger.Error().Err(err).Msg("failed to create keychain directory")
	}
	
	keyring, err := keychain.NewJSONKeyring(keyringFilePath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create JSON keyring")
		
		// Fallback to temporary file if home directory access fails
		tmpDir := os.TempDir()
		keyringFilePath = filepath.Join(tmpDir, "ipatool-auth.json")
		logger.Log().Msgf("Falling back to temporary storage: %s", keyringFilePath)
		
		keyring, err = keychain.NewJSONKeyring(keyringFilePath)
		if err != nil {
			// If we can't create a keyring at all, log error and exit
			logger.Error().Err(err).Msg("failed to create JSON keyring in temp directory")
			os.Exit(1)
		}
	}
	
	return keychain.New(keychain.Args{Keyring: keyring})
}

// initWithCommand initializes the dependencies of the command.
func initWithCommand(cmd *cobra.Command) {
	verbose := cmd.Flag("verbose").Value.String() == "true"
	interactive, _ := cmd.Context().Value("interactive").(bool)
	format := util.Must(OutputFormatFromString(cmd.Flag("format").Value.String()))

	dependencies.Logger = newLogger(format, verbose)
	dependencies.OS = operatingsystem.New()
	dependencies.Machine = machine.New(machine.Args{OS: dependencies.OS})
	dependencies.CookieJar = newCookieJar(dependencies.Machine)
	dependencies.Keychain = newKeychain(dependencies.Machine, dependencies.Logger, interactive)
	dependencies.AppStore = appstore.NewAppStore(appstore.Args{
		CookieJar:       dependencies.CookieJar,
		OperatingSystem: dependencies.OS,
		Keychain:        dependencies.Keychain,
		Machine:         dependencies.Machine,
	})

	util.Must("", createConfigDirectory(dependencies.OS, dependencies.Machine))
}

// createConfigDirectory creates the configuration directory for the CLI tool, if needed.
func createConfigDirectory(os operatingsystem.OperatingSystem, machine machine.Machine) error {
	configDirectoryPath := filepath.Join(machine.HomeDirectory(), ConfigDirectoryName)
	_, err := os.Stat(configDirectoryPath)

	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(configDirectoryPath, 0700)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("could not read metadata: %w", err)
	}

	return nil
}
