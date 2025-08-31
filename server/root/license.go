package root

// License命令结构体
type LicenseCmd struct {
	New    LicenseNewCmd    `command:"new" description:"Create a new license"`
	Delete LicenseDeleteCmd `command:"delete" description:"Delete a license"`
	Update LicenseUpdateCmd `command:"update" description:"Update a license"`
	List   LicenseListCmd   `command:"list" description:"List all licenses"`
	Get    LicenseGetCmd    `command:"get" description:"Get a license by id"`
}

type LicenseNewCmd struct {
	Username  string `long:"username" required:"true"`
	Email     string `long:"email" required:"true"`
	MaxBuilds int    `long:"max-builds" required:"true"`
	Days      int    `long:"days" required:"true"`
}

type LicenseDeleteCmd struct {
	ID string `long:"id" required:"true"`
}

type LicenseUpdateCmd struct {
	ID        string `long:"id" required:"true"`
	MaxBuilds int    `long:"max-builds"`
	Days      int    `long:"days"`
}

type LicenseListCmd struct{}

type LicenseGetCmd struct {
	ID string `long:"id" required:"true"`
}
