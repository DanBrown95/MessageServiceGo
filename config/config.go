package config

import (
	"fmt"
	"os"
	"strings"

	figgy "github.com/DanBrown95/go-figgy"
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
	SqlAddress                       string          `mapstructure:"SqlAddress"`
	MonitorAPIAddress                string          `mapstructure:"MonitorAPIAddress"`
	ProcessingPollingIntervalSeconds int             `mapstructure:"ProcessingPollingIntervalSeconds"`
	MessageSettings                  MessageSettings `mapstructure:"MessageSettings"`
}

var AppConfig Config

func getEnvOrDefault() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	return env
}

func loadSSMConfig(env string) {
	params := struct {
		Env string
	}{
		Env: strings.ToLower(env),
	}

	ssmClient := ssm.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})))
	secretManagerClient := secretsmanager.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})))

	err := figgy.LoadWithParameters(ssmClient, secretManagerClient, &AppConfig, params)
	if err != nil {
		println("Failed to load config from SSM/Secrets Manager:", err.Error())
		os.Exit(1)
	}
}

func LoadConfig() error {
	viper.SetConfigName("appsettings")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("fatal error reading default config: %w", err)
	}

	env := getEnvOrDefault()
	if env != "local" {
		viper.SetConfigName(fmt.Sprintf("appsettings.%s", strings.ToLower(env)))
		viper.AddConfigPath("./config")
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("fatal error reading config for %s: %w", env, err)
			}
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	loadSSMConfig(env)
	return nil
}
