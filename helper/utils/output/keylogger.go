package output

func ParseKeylogger(content []byte) (*KeyLoggerContext, error) {
	var keyloggerCtx *KeyLoggerContext
	keyloggerCtx = &KeyLoggerContext{
		Content: content,
	}
	return keyloggerCtx, nil
}
