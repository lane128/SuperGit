package lobedesk

// Persist semantic names independently from line thickness and node size.
func historyGraphStyleKey(key string) string {
	switch key {
	case "chamfer", "compact":
		return key
	}
	return "rounded"
}

func historyRefStyleKey(key string) string {
	if key == "stripe" {
		return key
	}
	return "segmented"
}
