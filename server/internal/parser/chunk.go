package parser

func Count(content []byte, max int) int {
	if max <= 0 {
		if len(content) == 0 {
			return 0
		}
		return 1
	}
	length := len(content)
	count := 0
	for i := 0; i < length; i += max {
		count++
	}
	return count
}

func Chunked(content []byte, max int) chan []byte {
	length := len(content)
	ch := make(chan []byte, 1)
	go func() {
		if max <= 0 {
			if length > 0 {
				ch <- content
			}
			close(ch)
			return
		}
		for i := 0; i < length; i += max {
			if i+max > length {
				ch <- content[i:]
			} else {
				ch <- content[i : i+max]
			}
		}
		close(ch)
	}()
	return ch
}
