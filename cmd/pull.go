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
	"github.com/Senetas/crypto-cli/crypto"
	"github.com/Senetas/crypto-cli/images"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull [OPTIONS] NAME[:TAG]",
	Short: "Download an image from a remote repository, decrypting if necessary.",
	Long: `pull is used to download an image from a repository, decrypt it if necessary and
load that images into the local docker engine. It is then avaliable to be run under the same
name as it was downloaded.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Flags().VisitAll(checkFlags)
		cryptotype, err := validateCryptoType(ctstr)
		if err != nil {
			return err
		}
		return runPull(args[0], passphrase, cryptotype)
	},
	Args: cobra.ExactArgs(1),
}

func runPull(remote, passphrase string, cryptotype crypto.EncAlgo) error {
	ref, err := reference.ParseNormalizedNamed(remote)
	if err != nil {
		return errors.Wrapf(err, "remote = ", remote)
	}

	if err = images.PullImage(ref, passphrase, cryptotype); err != nil {
		return errors.Wrapf(err, "ref = %v, cryptotype = %v", ref, cryptotype)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(pullCmd)

	pullCmd.Flags().StringVarP(&passphrase, "pass", "p", "", "Specifies the passphrase to use if passphrase encryption is selected")
	pullCmd.Flags().StringVarP(&ctstr, "type", "t", string(crypto.Pbkdf2Aes256Gcm), "Specifies the type of encryption to use.")
}
