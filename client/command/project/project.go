package project

import (
	"fmt"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
)

func ListProjectsCmd(cmd *cobra.Command, con *core.Console) error {
	projects, err := con.Rpc.ListProjects(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(projects.Projects) == 0 {
		con.Log.Console("No projects found\n")
		return nil
	}
	printProjects(con, projects)
	return nil
}

func CreateProjectCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	description, _ := cmd.Flags().GetString("description")
	note, _ := cmd.Flags().GetString("note")

	project, err := con.Rpc.CreateProject(con.Context(), &clientpb.CreateProjectRequest{
		Name:        name,
		Description: description,
		Note:        note,
	})
	if err != nil {
		return err
	}
	con.Log.Console(fmt.Sprintf("Project '%s' created (id: %s)\n", project.Name, project.Id))
	return nil
}

func GetProjectCmd(cmd *cobra.Command, con *core.Console) error {
	nameOrID := cmd.Flags().Arg(0)

	project, err := con.Rpc.GetProject(con.Context(), projectSelector(nameOrID))
	if err != nil {
		return err
	}
	printProjectDetail(con, project)
	return nil
}

func UpdateProjectCmd(cmd *cobra.Command, con *core.Console) error {
	nameOrID := cmd.Flags().Arg(0)

	existing, err := con.Rpc.GetProject(con.Context(), projectSelector(nameOrID))
	if err != nil {
		return err
	}

	newName, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	note, _ := cmd.Flags().GetString("note")

	project, err := con.Rpc.UpdateProject(con.Context(), &clientpb.UpdateProjectRequest{
		Id:          existing.Id,
		Name:        newName,
		Description: description,
		Note:        note,
	})
	if err != nil {
		return err
	}
	con.Log.Console(fmt.Sprintf("Project '%s' updated\n", project.Name))
	return nil
}

func DeleteProjectCmd(cmd *cobra.Command, con *core.Console) error {
	nameOrID := cmd.Flags().Arg(0)

	existing, err := con.Rpc.GetProject(con.Context(), projectSelector(nameOrID))
	if err != nil {
		return err
	}

	hard, _ := cmd.Flags().GetBool("hard")
	_, err = con.Rpc.DeleteProject(con.Context(), &clientpb.DeleteProjectRequest{
		Id:   existing.Id,
		Hard: hard,
	})
	if err != nil {
		return err
	}

	if hard {
		con.Log.Console(fmt.Sprintf("Project '%s' permanently deleted\n", existing.Name))
	} else {
		con.Log.Console(fmt.Sprintf("Project '%s' deleted (soft)\n", existing.Name))
	}
	return nil
}

func projectSelector(nameOrID string) *clientpb.Project {
	selector := &clientpb.Project{Name: nameOrID}
	if _, err := uuid.FromString(nameOrID); err == nil {
		selector.Id = nameOrID
		selector.Name = ""
	}
	return selector
}

func printProjects(con *core.Console, projects *clientpb.Projects) {
	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewFlexColumn("Description", "Description", 2),
		table.NewColumn("Created", "Created", 20),
	}, true)

	for _, p := range projects.Projects {
		rowEntries = append(rowEntries, table.NewRow(table.RowData{
			"Name":        p.Name,
			"Description": p.Description,
			"Created":     time.Unix(p.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		}))
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	tableModel.Title = "projects"
	con.Log.Console(tableModel.View())
}

func printProjectDetail(con *core.Console, p *clientpb.Project) {
	tui.RenderKVWithOptions(map[string]interface{}{
		"ID":          p.Id,
		"Name":        p.Name,
		"Description": p.Description,
		"Note":        p.Note,
		"Created":     time.Unix(p.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		"Updated":     time.Unix(p.UpdatedAt, 0).Format("2006-01-02 15:04:05"),
	}, []string{"ID", "Name", "Description", "Note", "Created", "Updated"}, tui.KVOptions{ShowHeader: true})
}
