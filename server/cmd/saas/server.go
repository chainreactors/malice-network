package main

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/assets"
	"github.com/chainreactors/malice-network/server/cmd/server"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/saas"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
	"os"
	"time"
)

func init() {
	err := configs.InitConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
	codenames.SetupCodenames()
	assets.SetupGithubFile()
}

func Execute() {
	var opt server.Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}
	if !fileutils.Exist(opt.Config) {
		confStr := configutil.InitDefaultConfig(&opt, 0)
		err := os.WriteFile(opt.Config, confStr, 0644)
		if err != nil {
			logs.Log.Errorf("cannot write default config , %s ", err.Error())
			return
		}
		logs.Log.Warnf("config file not found, created default config %s", opt.Config)
	}

	// 加载配置
	err = configutil.LoadConfig(opt.Config, &opt)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
		return
	}

	// 初始化证书
	err = certutils.GenerateRootCert()
	if err != nil {
		logs.Log.Errorf("cannot init root ca , %s ", err.Error())
		return
	}

	// 初始化数据库
	db.Client = db.NewDBClient()

	if parser.Active != nil {
		err = parseExecute(args, parser, &opt)
		if err != nil {
			logs.Log.Error(err)
		}
		return
	}

	// 创建Gin引擎
	r := gin.Default()

	// 设置路由
	saas.SetupRouter(r)

	// 获取saas配置
	saasConfig := configs.GetSaasConfig()
	addr := fmt.Sprintf("%s:%d", saasConfig.Host, saasConfig.Port)

	logs.Log.Infof("SaaS server starting on %s", addr)

	// 启动服务器
	err = r.Run(addr)
	if err != nil {
		logs.Log.Errorf("cannot start server , %s ", err.Error())
		return
	}
}

// parseExecute实现license命令分发
func parseExecute(args []string, parser *flags.Parser, opt *server.Options) error {
	if parser.Active == nil {
		return nil
	}
	switch parser.Active.Name {
	case "license":
		switch parser.Active.Active.Name {
		case "new":
			cmd := opt.License.New
			token, err := uuid.NewV4()
			if err != nil {
				return err
			}
			license := &models.License{
				Username:   cmd.Username,
				Email:      cmd.Email,
				Token:      token.String(),
				ExpireAt:   time.Now().Add(time.Duration(cmd.Days) * 24 * time.Hour),
				MaxBuilds:  cmd.MaxBuilds,
				BuildCount: 0,
			}
			if err := db.CreateLicense(license); err != nil {
				fmt.Println("Create license failed:", err)
				return err
			}
			fmt.Printf("License created: %+v\n", license)
			return nil
		case "delete":
			cmd := opt.License.Delete
			id, err := uuid.FromString(cmd.ID)
			if err != nil {
				fmt.Println("Invalid UUID:", err)
				return err
			}
			if err := db.DeleteLicenseByID(id); err != nil {
				fmt.Println("Delete license failed:", err)
				return err
			}
			fmt.Println("License deleted:", cmd.ID)
			return nil
		case "update":
			cmd := opt.License.Update
			id, err := uuid.FromString(cmd.ID)
			if err != nil {
				fmt.Println("Invalid UUID:", err)
				return err
			}
			license, err := db.GetLicenseByID(id)
			if err != nil {
				fmt.Println("License not found:", err)
				return err
			}
			if cmd.MaxBuilds > 0 {
				license.MaxBuilds = cmd.MaxBuilds
			}
			if cmd.Days > 0 {
				license.ExpireAt = time.Now().Add(time.Duration(cmd.Days) * 24 * time.Hour)
			}
			if err := db.UpdateLicense(license); err != nil {
				fmt.Println("Update license failed:", err)
				return err
			}
			fmt.Printf("License updated: %+v\n", license)
			return nil
		case "list":
			licenses, err := db.ListLicenses()
			if err != nil {
				fmt.Println("List licenses failed:", err)
				return err
			}
			for _, l := range licenses {
				fmt.Printf("%+v\n", l)
			}
			return nil
		case "get":
			cmd := opt.License.Get
			id, err := uuid.FromString(cmd.ID)
			if err != nil {
				fmt.Println("Invalid UUID:", err)
				return err
			}
			license, err := db.GetLicenseByID(id)
			if err != nil {
				fmt.Println("License not found:", err)
				return err
			}
			fmt.Printf("%+v\n", license)
			return nil
		}
	}
	return errors.New("unknown command")
}

func main() {
	Execute()
}
