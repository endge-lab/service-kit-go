package config

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func LoadServiceConfig() (*ServiceConfig, error) {
	// Сначала пытаемся загрузить .env-файлы.
	// Это нужно, чтобы переменные из .env попали в os.Getenv до настройки Viper.
	preloadServiceEnvFiles()

	// Определяем окружение: development/local/production/etc.
	// Если APP_ENV/GO_ENV/ENVIRONMENT/NODE_ENV не заданы,
	// будет defaultServiceAppEnv, скорее всего "development".
	appEnv := NormalizeServiceEnvironment(detectServiceAppEnv())

	// Создаём отдельный экземпляр Viper.
	// Это лучше, чем использовать глобальный viper, потому что нет shared state.
	v := viper.New()

	// Говорим Viper, что конфиг будет YAML.
	v.SetConfigType("yaml")

	// Позволяет мапить ключи вида postgres.host на env POSTGRES_HOST.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Разрешаем Viper читать переменные окружения автоматически.
	v.AutomaticEnv()

	// Ставим дефолтные значения конфига.
	// Они будут использованы, если нет значения ни в yaml, ни в env.
	setServiceDefaults(v, appEnv)

	// Привязываем конкретные ключи конфига к env-переменным.
	if err := bindServiceEnv(v); err != nil {
		return nil, err
	}

	// Выбираем файл конфига.
	// Например для development будет configs/development.yaml.
	// Если задан CONFIG_PATH, берём конкретно его.
	setServiceConfigFile(v, appEnv)

	// Читаем YAML-файл.
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Раскладываем значения из Viper в структуру ServiceConfig.
	// DecodeHook нужен, чтобы строки типа "5s", "1m" превращались в time.Duration.
	var cfg ServiceConfig
	if err := v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc())); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Нормализуем конфиг после чтения.
	// Например привести env к lowercase, заполнить derived fields, trim строк и т.д.
	cfg.Normalize()

	// Проверяем, что конфиг валидный.
	// Например обязательные поля, порты, URL, Postgres credentials.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
