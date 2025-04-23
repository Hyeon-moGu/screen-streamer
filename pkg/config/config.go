package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	RtpPayloadMaxSize int    `mapstructure:"rtpPayloadMaxSize"`
	ResizeWidth       uint   `mapstructure:"resizeWidth"`
	ResizeHeight      uint   `mapstructure:"resizeHeight"`
	JpegQuality       int    `mapstructure:"jpegQuality"`
	FrameRate         int    `mapstructure:"frameRate"`
	DebugYn           string `mapstructure:"debugYn"`
	RtspPort          int    `mapstructure:"rtspPort"`
	RtspPath          string `mapstructure:"rtspPath"`
	DisplayIndex      int    `mapstructure:"displayIndex"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	var cfg Config
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
