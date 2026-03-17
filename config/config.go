package config

import (
	"fmt"
	"os"
	"strings"

	figgy "github.com/DanBrown95/go-figgy"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/viper"
)

type MessageSettings struct {
	EncryptionKey string `secretmanager:"rerolldrinks/message/encryption/{{.Env}}"`
	RequestKey    string `secretmanager:"rerolldrinks/message/api-request-key/{{.Env}}"`
}

type Config struct {
	SqlAddress                       string          `mapstructure:"SqlAddress" ssm:"/rerolldrinks/{{.Env}}/sql/conn"`
	MonitorAPIAddress                string          `mapstructure:"MonitorAPIAddress"`
	ProcessingPollingIntervalSeconds int             `mapstructure:"ProcessingPollingIntervalSeconds"`
	AWSRegion                        string          `mapstructure:"AWSRegion"`
	MessageSettings                  MessageSettings `mapstructure:"MessageSettings"`
}

var AppConfig Config

func getEnvOrDefault() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}
	return env
}

func loadSSMConfig(env string) {
	ssmEnv := strings.ToLower(env)
	if ssmEnv == "local" {
		ssmEnv = "development"
	}

	params := struct {
		Env string
	}{
		Env: ssmEnv,
	}

	ssmClient := ssm.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String(AppConfig.AWSRegion)},
	})))
	secretManagerClient := secretsmanager.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String(AppConfig.AWSRegion)},
	})))

	localSqlAddress := AppConfig.SqlAddress

	err := figgy.LoadWithParameters(ssmClient, secretManagerClient, &AppConfig, params)
	if err != nil {
		println("Failed to load config from SSM/Secrets Manager:", err.Error())
		os.Exit(1)
	}

	if env == "local" {
		AppConfig.SqlAddress = localSqlAddress
	}
}

func LoadConfig() error {
	viper.SetConfigName("appsettings")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("fatal error reading default config: %w", err)
	}

	env := getEnvOrDefault()

	viper.SetConfigName(fmt.Sprintf("appsettings.%s", strings.ToLower(env)))
	viper.AddConfigPath("./config")
	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("fatal error reading config for %s: %w", env, err)
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	loadSSMConfig(env)
	return nil
}
