package project

import (
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	projectCmd := &cobra.Command{
		Use:   consts.CommandProject,
		Short: "Manage projects",
		Long:  "List, create, update, and delete projects.",
		Example: `~~~
project
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListProjectsCmd(cmd, con)
		},
	}

	createCmd := &cobra.Command{
		Use:   consts.CommandProjectCreate + " [name]",
		Short: "Create a new project",
		Long:  "Create a project with an optional description and note.",
		Example: `~~~
project create red-team-q3 --description "Q3 assessment" --note "Internal"
~~~`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("project name is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateProjectCmd(cmd, con)
		},
	}
	common.BindFlag(createCmd, func(f *pflag.FlagSet) {
		f.StringP("description", "d", "", "project description")
		f.StringP("note", "n", "", "project note")
	})

	getCmd := &cobra.Command{
		Use:   consts.CommandProjectGet + " [name|id]",
		Short: "Show project details",
		Long:  "Show project details by project name or ID.",
		Example: `~~~
project get red-team-q3
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetProjectCmd(cmd, con)
		},
	}

	updateCmd := &cobra.Command{
		Use:   consts.CommandProjectUpdate + " [name|id]",
		Short: "Update a project",
		Long:  "Update a project's name, description, or note.",
		Example: `~~~
project update red-team-q3 --name red-team-q3-final
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateProjectCmd(cmd, con)
		},
	}
	common.BindFlag(updateCmd, func(f *pflag.FlagSet) {
		f.String("name", "", "new project name")
		f.StringP("description", "d", "", "new description")
		f.StringP("note", "n", "", "new note")
	})

	deleteCmd := &cobra.Command{
		Use:   consts.CommandProjectDelete + " [name|id]",
		Short: "Delete a project",
		Long:  "Soft-delete a project, or permanently delete it with --hard.",
		Example: `~~~
project delete red-team-q3
project delete red-team-q3 --hard
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteProjectCmd(cmd, con)
		},
	}
	common.BindFlag(deleteCmd, func(f *pflag.FlagSet) {
		f.Bool("hard", false, "permanently delete (cannot be undone)")
	})

	projectCmd.AddCommand(createCmd, getCmd, updateCmd, deleteCmd)

	return []*cobra.Command{projectCmd}
}
