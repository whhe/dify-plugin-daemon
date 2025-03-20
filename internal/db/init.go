package db

import (
	"strings"

	"github.com/langgenius/dify-plugin-daemon/internal/types/app"
	"github.com/langgenius/dify-plugin-daemon/internal/types/models"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"gorm.io/gorm"
)

type DBInitializer interface {
	Connect(dbName string) (*gorm.DB, error)
	CreateDatabaseIfNotExists(db *gorm.DB, dbName string) error
	Setup(db *gorm.DB) error
}

func initDifyPluginDB(dbInitializer DBInitializer, db_name string, default_db_name string) error {
	// first try to connect to target database
	db, err := dbInitializer.Connect(db_name)
	if err != nil {
		// if connection fails, try to create database
		db, err = dbInitializer.Connect(default_db_name)
		if err != nil {
			return err
		}

		err = dbInitializer.CreateDatabaseIfNotExists(db, db_name)
		if err != nil {
			return err
		}

		// connect to the new db
		db, err = dbInitializer.Connect(db_name)
		if err != nil {
			return err
		}
	}

	err = dbInitializer.Setup(db)
	if err != nil {
		return err
	}

	DifyPluginDB = db

	return nil
}

func autoMigrate() error {
	err := DifyPluginDB.AutoMigrate(
		models.Plugin{},
		models.PluginInstallation{},
		models.PluginDeclaration{},
		models.Endpoint{},
		models.ServerlessRuntime{},
		models.ToolInstallation{},
		models.AIModelInstallation{},
		models.InstallTask{},
		models.TenantStorage{},
		models.AgentStrategyInstallation{},
	)

	if err != nil {
		return err
	}

	// check if "declaration" table exists in Plugin/ServerlessRuntime/ToolInstallation/AIModelInstallation/AgentStrategyInstallation
	// delete the column if exists
	ignoreDeclarationColumn := func(table string) error {
		if DifyPluginDB.Migrator().HasColumn(table, "declaration") {
			// remove NOT NULL constraint on declaration column
			if err := DifyPluginDB.Exec("ALTER TABLE " + table + " ALTER COLUMN declaration DROP NOT NULL").Error; err != nil {
				return err
			}
		}
		return nil
	}

	if err := ignoreDeclarationColumn("plugins"); err != nil {
		return err
	}

	if err := ignoreDeclarationColumn("serverless_runtimes"); err != nil {
		return err
	}

	if err := ignoreDeclarationColumn("tool_installations"); err != nil {
		return err
	}

	if err := ignoreDeclarationColumn("ai_model_installations"); err != nil {
		return err
	}

	if err := ignoreDeclarationColumn("agent_strategy_installations"); err != nil {
		return err
	}

	return nil
}

func Init(config *app.Config) {
	var (
		dbInitializer DBInitializer
		err           error
	)
	if config.SQLAlchemyDatabaseURIScheme == "postgresql" {
		dbInitializer = NewPostgresInitializer(config.DBHost,
			int(config.DBPort),
			config.DBUsername,
			config.DBPassword,
			config.DBSslMode)
	} else if strings.Contains(config.SQLAlchemyDatabaseURIScheme, "mysql") {
		dbInitializer = NewMySQLInitializer(config.DBHost,
			int(config.DBPort),
			config.DBUsername,
			config.DBPassword,
			config.DBSslMode)
	} else {
		log.Panic("unsupported uri scheme: %v", config.SQLAlchemyDatabaseURIScheme)
	}

	err = initDifyPluginDB(dbInitializer, config.DBDatabase, config.DBDefaultDatabase)

	if err != nil {
		log.Panic("failed to init dify plugin db: %v", err)
	}

	err = autoMigrate()
	if err != nil {
		log.Panic("failed to auto migrate: %v", err)
	}

	log.Info("dify plugin db initialized")
}

func Close() {
	db, err := DifyPluginDB.DB()
	if err != nil {
		log.Error("failed to close dify plugin db: %v", err)
		return
	}

	err = db.Close()
	if err != nil {
		log.Error("failed to close dify plugin db: %v", err)
		return
	}

	log.Info("dify plugin db closed")
}
