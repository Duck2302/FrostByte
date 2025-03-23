package main

func splitFile(file []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(file); i += chunkSize {
		end := i + chunkSize
		if end > len(file) {
			end = len(file)
		}
		chunk := file[i:end]
		chunks = append(chunks, chunk)
	}
	return chunks
}
