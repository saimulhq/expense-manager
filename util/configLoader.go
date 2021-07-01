package util

import "github.com/spf13/viper"

type Config struct {
	DBUrl            string `mapstructure:"dbUrl"`
	DBName           string `mapstructure:"dbName"`
	DBCollectionName string `mapstructure:"dbCollection"`
	GrpcHost         string `mapstructure:"grpcHost"`
	GrpcPort         string `mapstructure:"grpcPort"`
	HttpHost         string `mapstructure:"httpHost"`
	HttpPort         string `mapstructure:"httpPort"`
}

// function to load config from config.json
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
