package assets

import "embed"

var (
	//go:embed fs/windows/*
	assetsFs embed.FS
)
