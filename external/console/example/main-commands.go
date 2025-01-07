package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/console"
	"github.com/reeflective/console/commands/readline"
)

// mainMenuCommands - Create the commands for the main menu.
// Most of these commands have an empty implementation, and all
// have been generated with ChatGPT prompts.
func mainMenuCommands(app *console.Console) console.Commands {
	return func() *cobra.Command {
		rootCmd := &cobra.Command{}
		rootCmd.Short = shortUsage

		rootCmd.AddGroup(
			&cobra.Group{ID: "core", Title: "core"},
			&cobra.Group{ID: "filesystem", Title: "filesystem"},
			&cobra.Group{ID: "deployment", Title: "deployment"},
			&cobra.Group{ID: "tools", Title: "tools"},
		)

		// Readline subcommands
		rootCmd.AddCommand(readline.Commands(app.Shell()))

		exitCmd := &cobra.Command{
			Use:     "exit",
			Short:   "Exit the console application",
			GroupID: "core",
			Run: func(cmd *cobra.Command, args []string) {
				exitCtrlD(app)
			},
		}
		rootCmd.AddCommand(exitCmd)

		// And let's add a command declared in a traditional "cobra" way.
		clientMenuCommand := &cobra.Command{
			Use:     "client",
			Short:   "Switch to the client menu (also works with CtrlC)",
			GroupID: "core",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Switching to client menu")
				app.SwitchMenu("client")
			},
		}
		rootCmd.AddCommand(clientMenuCommand)

		helloCmd := &cobra.Command{
			Use:     "hello",
			Short:   "Say hello with customizable message",
			GroupID: "core",
			Run: func(cmd *cobra.Command, args []string) {
				// This is the implementation logic for the hello command.
				message, _ := cmd.Flags().GetString("message")
				count, _ := cmd.Flags().GetInt("count")

				for i := 0; i < count; i++ {
					fmt.Println(message)
				}
			},
		}

		// Add flags to the hello command
		helloCmd.Flags().String("message", "Hello, World!", "Customize the greeting message")
		helloCmd.Flags().Int("count", 1, "Number of times to print the message")

		// Add the hello command as a subcommand of the root command
		rootCmd.AddCommand(helloCmd)

		greetCmd := &cobra.Command{
			Use:     "greet",
			Short:   "Greet a person",
			GroupID: "core",
			Run: func(cmd *cobra.Command, args []string) {
				name, _ := cmd.Flags().GetString("name")
				age, _ := cmd.Flags().GetInt("age")

				fmt.Printf("Hello, %s! You are %d years old.\n", name, age)
			},
		}

		greetCmd.Flags().String("name", "", "Specify a name to greet")
		greetCmd.Flags().Int("age", 0, "Specify the age of the person")

		rootCmd.AddCommand(greetCmd)

		convertCmd := &cobra.Command{
			Use:     "convert",
			Short:   "Convert a file",
			GroupID: "filesystem",
			Run: func(cmd *cobra.Command, args []string) {
				inputFile, _ := cmd.Flags().GetString("input")
				outputFile, _ := cmd.Flags().GetString("output")
				appendMode, _ := cmd.Flags().GetBool("append")

				fmt.Printf("Converting file: %s\n", inputFile)
				fmt.Printf("Output file: %s\n", outputFile)

				if appendMode {
					fmt.Println("Append mode: ON")
				} else {
					fmt.Println("Append mode: OFF")
				}
			},
		}

		convertCmd.Flags().String("input", "", "Specify the input file")
		convertCmd.Flags().String("output", "", "Specify the output file")
		convertCmd.Flags().Bool("append", false, "Enable append mode")

		rootCmd.AddCommand(convertCmd)

		mkdirCmd := &cobra.Command{
			Use:     "mkdir [flags] DIRECTORY...",
			Short:   "Create directories",
			GroupID: "filesystem",
			Args:    cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				verbose, _ := cmd.Flags().GetBool("verbose")
				parents, _ := cmd.Flags().GetBool("parents")

				for _, dir := range args {
					var err error
					if parents {
						err = os.MkdirAll(dir, os.ModePerm)
					} else {
						err = os.Mkdir(dir, os.ModePerm)
					}

					if err != nil {
						fmt.Printf("Error creating directory: %s\n", err)
					} else if verbose {
						fmt.Printf("Created directory: %s\n", dir)
					}
				}
			},
		}

		mkdirCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		mkdirCmd.Flags().BoolP("parents", "p", false, "Make parent directories as needed")
		rootCmd.AddCommand(mkdirCmd)

		lsCmd := &cobra.Command{
			Use:     "ls [flags] [DIRECTORY]",
			Short:   "List directory contents",
			GroupID: "filesystem",
			Args:    cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				directory := "./"
				if len(args) > 0 {
					directory = args[0]
				}

				fmt.Println("Running ls command with directory:", directory)

				// Implementation logic for ls command
				// Customize or extend the logic as needed
			},
		}

		lsCmd.Flags().BoolP("long", "l", false, "Use a long listing format")
		lsCmd.Flags().Bool("human-readable", false, "Print sizes in human-readable format")
		lsCmd.Flags().Bool("all", false, "Do not ignore entries starting with .")
		lsCmd.Flags().Bool("recursive", false, "List subdirectories recursively")
		lsCmd.Flags().Bool("hidden", false, "Show hidden files")
		lsCmd.Flags().Bool("sort-by-size", false, "Sort by file size")
		lsCmd.Flags().Bool("sort-by-time", false, "Sort by modification time")
		lsCmd.Flags().Bool("reverse", false, "Reverse order while sorting")
		rootCmd.AddCommand(lsCmd)

		sshCmd := &cobra.Command{
			Use:     "ssh [flags] USER@HOST",
			Short:   "SSH client",
			GroupID: "tools",
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				userHost := args[0]

				identityFile, _ := cmd.Flags().GetString("identity")
				port, _ := cmd.Flags().GetString("port")
				verbose, _ := cmd.Flags().GetBool("verbose")

				sshArgs := []string{"-l", userHost}
				if identityFile != "" {
					sshArgs = append(sshArgs, "-i", identityFile)
				}
				if port != "" {
					sshArgs = append(sshArgs, "-p", port)
				}

				sshArgs = append(sshArgs, "echo", "Hello, SSH!")

				sshCmd := exec.Command("ssh", sshArgs...)

				if verbose {
					fmt.Println("Executing SSH command:", strings.Join(sshCmd.Args, " "))
				}

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				err := sshCmd.Run()
				if err != nil {
					fmt.Printf("SSH command failed: %s\n", err)
					os.Exit(1)
				}
			},
		}

		sshCmd.Flags().String("identity", "", "Specify the identity file")
		sshCmd.Flags().String("port", "", "Specify the SSH port")
		sshCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		rootCmd.AddCommand(sshCmd)

		gitCmd := &cobra.Command{
			Use:     "git [flags] <command>",
			Short:   "Git command",
			GroupID: "tools",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Git command invoked without a specific subcommand")
				cmd.Usage()
			},
		}
		rootCmd.AddCommand(gitCmd)

		cloneCmd := &cobra.Command{
			Use:   "clone REPO_URL [DESTINATION]",
			Short: "Clone a repository",
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				repoURL := args[0]
				destination := "./"
				if len(args) == 2 {
					destination = args[1]
				}

				fmt.Printf("Cloning repository: %s to %s\n", repoURL, destination)

				gitArgs := []string{"clone", repoURL, destination}

				gitCmd := exec.Command("git", gitArgs...)
				gitCmd.Stdout = os.Stdout
				gitCmd.Stderr = os.Stderr

				err := gitCmd.Run()
				if err != nil {
					fmt.Printf("Git clone failed: %s\n", err)
					os.Exit(1)
				}
			},
		}

		cloneCmd.Flags().StringP("branch", "b", "", "Checkout a specific branch")
		cloneCmd.Flags().Bool("bare", false, "Create a bare repository")

		gitCmd.AddCommand(cloneCmd)

		checkoutCmd := &cobra.Command{
			Use:   "checkout [flags] [BRANCH]",
			Short: "Switch branches or restore working tree files",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if len(args) == 0 {
					// Default behavior, print help message
					cmd.Help()
					return
				}

				branch := args[0]

				// Implementation logic for git checkout command

				// Customize or extend the logic as needed

				fmt.Println("Running git checkout command with branch:", branch)
			},
		}

		checkoutCmd.Flags().BoolP("force", "f", false, "Force checkout")
		checkoutCmd.Flags().BoolP("create", "b", false, "Create and checkout a new branch")
		gitCmd.AddCommand(checkoutCmd)

		commitCmd := &cobra.Command{
			Use:   "commit [flags]",
			Short: "Record changes to the repository",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Running git commit command")

				// Implementation logic for git commit command

				// Customize or extend the logic as needed
			},
		}

		commitCmd.Flags().StringP("message", "m", "", "Commit message")
		commitCmd.Flags().Bool("amend", false, "Amend the previous commit")
		gitCmd.AddCommand(commitCmd)

		pullCmd := &cobra.Command{
			Use:   "pull [flags]",
			Short: "Fetch from and integrate with another repository or a local branch",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Running git pull command")

				// Implementation logic for git pull command

				// Customize or extend the logic as needed
			},
		}

		pullCmd.Flags().String("rebase", "", "Rebase local branch onto fetched branch")
		gitCmd.AddCommand(pullCmd)

		pushCmd := &cobra.Command{
			Use:   "push [flags]",
			Short: "Update remote refs along with associated objects",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Running git push command")

				// Implementation logic for git push command

				// Customize or extend the logic as needed
			},
		}

		pushCmd.Flags().BoolP("force", "f", false, "Force push")
		pushCmd.Flags().Bool("tags", false, "Push tags")

		// Add more flags or logic as needed
		gitCmd.AddCommand(pushCmd)

		downloadCmd := &cobra.Command{
			Use:     "download [flags] URL DESTINATION",
			Short:   "Download a file from a URL",
			GroupID: "filesystem",
			Args:    cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				url := args[0]
				destination := args[1]

				// Implementation logic for download command

				// Customize or extend the logic as needed

				fmt.Printf("Downloading file from URL: %s to destination: %s\n", url, destination)
			},
		}

		downloadCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		downloadCmd.Flags().StringP("user-agent", "u", "", "Set the User-Agent header")
		rootCmd.AddCommand(downloadCmd)

		encryptCmd := &cobra.Command{
			Use:     "encrypt [flags] FILE",
			Short:   "Encrypt a file",
			GroupID: "filesystem",
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]

				// Implementation logic for encrypt command

				// Customize or extend the logic as needed

				fmt.Println("Encrypting file:", file)
			},
		}

		encryptCmd.Flags().StringP("algorithm", "a", "aes256", "Set the encryption algorithm")
		encryptCmd.Flags().BoolP("force", "f", false, "Force encryption")
		encryptCmd.Flags().StringP("key", "k", "", "Specify the encryption key")
		encryptCmd.Flags().StringP("output", "o", "", "Specify the output file")

		rootCmd.AddCommand(encryptCmd)

		searchCmd := &cobra.Command{
			Use:     "search [flags] QUERY",
			Short:   "Search for a query",
			GroupID: "filesystem",
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				query := args[0]

				// Implementation logic for search command

				// Customize or extend the logic as needed

				fmt.Println("Searching for query:", query)
			},
		}
		searchCmd.Flags().BoolP("case-sensitive", "c", false, "Perform case-sensitive search")
		searchCmd.Flags().BoolP("regex", "r", false, "Interpret the query as a regular expression")
		searchCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")

		rootCmd.AddCommand(searchCmd)

		backupCmd := &cobra.Command{
			Use:     "backup [flags] SOURCE DESTINATION",
			Short:   "Create a backup of a file or directory",
			GroupID: "filesystem",
			Args:    cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				source := args[0]
				destination := args[1]

				// Implementation logic for backup command

				fmt.Printf("Creating backup of %s to %s\n", source, destination)
			},
		}

		backupCmd.Flags().BoolP("incremental", "i", false, "Perform incremental backup")
		backupCmd.Flags().StringP("compression", "c", "gzip", "Specify the compression algorithm")
		backupCmd.Flags().Bool("dry-run", false, "Perform a dry run without actually creating the backup")
		rootCmd.AddCommand(backupCmd)

		renameCmd := &cobra.Command{
			Use:     "rename [flags] FILE NEW_NAME",
			Short:   "Rename a file",
			GroupID: "filesystem",
			Args:    cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]
				newName := args[1]

				// Implementation logic for rename command

				fmt.Printf("Renaming file %s to %s\n", file, newName)
			},
		}

		renameCmd.Flags().BoolP("force", "f", false, "Force rename, even if the new name already exists")
		renameCmd.Flags().BoolP("preserve-extension", "p", false, "Preserve the file extension while renaming")

		rootCmd.AddCommand(renameCmd)

		deployCmd := &cobra.Command{
			Use:     "deploy [flags] FILE",
			Short:   "Deploy a file",
			GroupID: "deployment",
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]

				// Implementation logic for deploy command

				fmt.Println("Deploying file:", file)
			},
		}

		deployCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		deployCmd.Flags().StringP("target", "t", "", "Specify the deployment target")
		deployCmd.Flags().Bool("clean", false, "Perform a clean deployment, removing previous versions")

		rootCmd.AddCommand(deployCmd)

		deployWebCmd := &cobra.Command{
			Use:   "web [flags]",
			Short: "Deploy a web application",
			Run: func(cmd *cobra.Command, args []string) {
				// Implementation logic for deploying a web application

				fmt.Println("Deploying web application")
			},
		}

		deployCmd.AddCommand(deployWebCmd)

		deployAPICmd := &cobra.Command{
			Use:   "api [flags]",
			Short: "Deploy an API service",
			Run: func(cmd *cobra.Command, args []string) {
				// Implementation logic for deploying an API service

				fmt.Println("Deploying API service")
			},
		}

		deployCmd.AddCommand(deployAPICmd)

		deployDatabaseCmd := &cobra.Command{
			Use:   "database [flags]",
			Short: "Deploy a database",
			Run: func(cmd *cobra.Command, args []string) {
				// Implementation logic for deploying a database

				fmt.Println("Deploying database")
			},
		}

		deployCmd.AddCommand(deployDatabaseCmd)

		localCmd := &cobra.Command{
			Use:   "local [flags] FILE",
			Short: "Deploy a file locally",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]

				// Implementation logic for local subcommand

				fmt.Println("Deploying file locally:", file)
			},
		}

		localCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")

		deployCmd.AddCommand(localCmd)

		remoteCmd := &cobra.Command{
			Use:   "remote [flags] FILE",
			Short: "Deploy a file remotely",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]

				// Implementation logic for remote subcommand

				fmt.Println("Deploying file remotely:", file)
			},
		}

		remoteCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		remoteCmd.Flags().StringP("host", "h", "", "Specify the remote host")

		deployCmd.AddCommand(remoteCmd)

		cloudCmd := &cobra.Command{
			Use:   "cloud [flags] FILE",
			Short: "Deploy a file to the cloud",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				file := args[0]

				// Implementation logic for cloud subcommand

				fmt.Println("Deploying file to the cloud:", file)
			},
		}

		cloudCmd.Flags().BoolP("verbose", "v", false, "Print verbose output")
		cloudCmd.Flags().StringP("provider", "p", "", "Specify the cloud provider")

		deployCmd.AddCommand(cloudCmd)

		//
		// Completions ----------------------------------------------------------------- //
		//

		// For each of the commands above, generate the carapace.Carapace for the command.
		// Then create a map carapace.FlagMap, and add file completion to all flags requiring
		// a file argument.
		for _, cmd := range rootCmd.Commands() {
			c := carapace.Gen(cmd)

			if cmd.Args != nil {
				c.PositionalAnyCompletion(
					carapace.ActionCallback(func(c carapace.Context) carapace.Action {
						return carapace.ActionFiles()
					}),
				)
			}

			flagMap := make(carapace.ActionMap)
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if f.Name == "file" || strings.Contains(f.Usage, "file") {
					flagMap[f.Name] = carapace.ActionFiles()
				}
			})

			if cmd.Name() == "ssh" {
				// Generate a list of random hosts to use as positional arguments
				hosts := make([]string, 0)
				for i := 0; i < 10; i++ {
					hosts = append(hosts, fmt.Sprintf("host%d", i))
				}
				c.PositionalCompletion(carapace.ActionValues(hosts...))
			}

			if cmd.Name() == "encrypt" {
				cmd.Flags().VisitAll(func(f *pflag.Flag) {
					if f.Name == "algorithm" {
						flagMap[f.Name] = carapace.ActionValues("aes", "des", "blowfish")
					}
				})
			}

			c.FlagCompletion(flagMap)
		}

		rootCmd.SetHelpCommandGroupID("core")
		rootCmd.InitDefaultHelpCmd()
		rootCmd.CompletionOptions.DisableDefaultCmd = true
		rootCmd.DisableFlagsInUseLine = true

		return rootCmd
	}
}
