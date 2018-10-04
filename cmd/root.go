// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/google/go-github/github"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"log"
	"os"
	"path"
	"strings"

	_ "github.com/joho/godotenv/autoload"

	oauth2ns "github.com/nmrshll/oauth2-noserver"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/spf13/cobra"
)

var (
	// You must register the app at https://github.com/settings/applications
	// Set callback to http://127.0.0.1:7000/github_oauth_cb
	// Set ClientId and ClientSecret to
	oauthConf *oauth2.Config

	ctx = context.Background()
)

func init() {
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("HACKTOBERFEST_CLIENT_ID"),
		ClientSecret: os.Getenv("HACKTOBERFEST_CLIENT_SECRET"),
		Scopes:       []string{"repo"},
		Endpoint:     githuboauth.Endpoint,
	}

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hacktoberfest-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("state", "s", "", "specify the state of the prs you want to see")
}

func loadToken() *oauth2.Token {
	hd, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	fp := path.Join(hd, ".config", "hacktoberfest-cli")
	tokenPath := path.Join(fp, ".token.json")

	file, err := os.Open(tokenPath)
	if err == nil {
		res := new(oauth2.Token)
		decoder := json.NewDecoder(file)
		err := decoder.Decode(res)

		if err != nil {
			panic(err)
		}

		return res
	}

	cl, err := oauth2ns.AuthenticateUser(oauthConf)
	if err != nil {
		log.Fatalln(err)
	}

	err = os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err = os.Create(tokenPath)
	if err != nil {
		panic(err)
	}

	saver := json.NewEncoder(file)
	saver.Encode(cl.Token)

	return cl.Token
}

func client() *github.Client {
	token := loadToken()
	ts := oauth2.StaticTokenSource(token)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hacktoberfest-cli",
	Short: "See your contributions to Hacktoberfest without leaving your terminal",
	Run: func(cmd *cobra.Command, args []string) {
		client := client()

		// october := time.Now().Month()
		un := "tomlazar"
		ss := fmt.Sprintf("author:%s created:2018-09-30..2018-11-01 type:pr", un)

		if st, err := cmd.Flags().GetString("state"); err == nil && st != "" {
			ss = fmt.Sprintf("%v state:%v", ss, st)
		}

		res, _, err := client.Search.Issues(ctx, ss, nil)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Total: %d\n", res.GetTotal())
		for _, x := range res.Issues {

			split := strings.Split(*x.RepositoryURL, "/")
			l := len(split)

			pr, _, err := client.PullRequests.Get(ctx, split[l-2], split[l-1], *x.Number)

			c := color.New(color.FgHiYellow)
			if *pr.Merged {
				c = color.New(color.FgHiMagenta)
			} else if *pr.State == "closed" {
				c = color.New(color.FgHiRed)
			} else if *pr.State == "open" {
				c = color.New(color.FgHiGreen)
			}

			if err != nil {
				log.Fatal(err)
			}

			title := strings.Join([]string{split[l-2], split[l-1]}, "/")

			fmt.Printf("\t%40v: %v\n", c.Sprint(title), *x.Title)
		}
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".hacktoberfest-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".hacktoberfest-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
