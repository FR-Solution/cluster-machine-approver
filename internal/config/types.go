package config

type Config struct {
	KubeconfigPath     string `yaml:"kubeconfig_path"`
	AIMJson            []byte `yaml:"aim_json"`
	FolderID           string `yaml:"folder_id"`
	InstanceNameLayout string `yaml:"instance_name_layout"`
}
