// Copyright © 2018 SENETAS SECURITY PTY LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Senetas/crypto-cli/crypto"
	"github.com/Senetas/crypto-cli/utils"
)

var (
	typeStr    string
	tempDir    string
	passphrase string
	debug      bool
	opts       = crypto.Opts{
		Algos:  crypto.Pbkdf2Aes256Gcm,
		Compat: false,
	}

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "crypto-cli [OPTIONS] [command]",
		Short: "A command line utility to encrypt and decrypt docker images and store them in docker registries",
		Long: `Crypto-Cli is a command line utility to encrypt and decrypt docker images and stores
them in repositories online. It maybe used to distribute docker images
confidentially. It does not sign images so cannot guarantee identities.

The operations emulate docker push and docker pull but will encrypt then
MAC the images before uploading them, and check the MAC then decrypt after
downloading them.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		c, ok := errors.Cause(err).(utils.Error)
		if debug && (!ok || c.HasStack) {
			log.Fatal().Msgf("%+v", err)
		}
		log.Fatal().Msgf("%v", err)
	}
}

func init() {
	// use a prettier logger, <nil> timestamp
	log.Logger = zerolog.New(ConsoleWriter{Out: os.Stderr}).With().Logger()

	cobra.OnInitialize(initLogging)

	rootCmd.PersistentFlags().StringVarP(
		&passphrase,
		"pass",
		"p",
		"",
		`Specifies the passphrase to use for encryption or decryption as applicable.
If absent, a prompt will be presented.`,
	)

	rootCmd.PersistentFlags().BoolVarP(
		&debug,
		"verbose",
		"v",
		false,
		"Set the log level to debug",
	)

	rootCmd.PersistentFlags().StringVar(
		&tempDir,
		"temp",
		filepath.Join(os.TempDir(), "com.senetas.crypto"),
		`Specifies the directory to store temporary files.`,
	)
}

func initLogging() {
	// hide debug logs by default
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
