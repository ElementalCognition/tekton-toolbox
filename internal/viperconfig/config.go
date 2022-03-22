package viperconfig

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewConfig(component string, flags *pflag.FlagSet) (*viper.Viper, error) {
	v := viper.New()
	err := v.BindPFlags(flags)
	if err != nil {
		return nil, err
	}
	v.SetConfigType("yaml")
	c := v.GetString("config")
	if len(c) > 0 {
		v.SetConfigFile(c)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(fmt.Sprintf("$HOME/.config/%s", component))
		v.AddConfigPath(fmt.Sprintf("/etc/config/%s", component))
		v.AddConfigPath(fmt.Sprintf("./config/%s", component))
	}
	v.AutomaticEnv()
	return v, nil
}

func LoadConfig(v *viper.Viper, cfg interface{}) error {
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	err = v.Unmarshal(&cfg)
	return err
}
