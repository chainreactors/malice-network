package assets

import "embed"

var (
	//go:embed  fs/linux/*
	assetsFs embed.FS
)
