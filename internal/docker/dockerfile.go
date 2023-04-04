package docker

import (
	"bufio"
	"io"
	"strings"
)

func ParseDockerfileBuildStages(dockerfile io.Reader) ([]string, error) {
	s := bufio.NewScanner(dockerfile)
	var stages []string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if !strings.EqualFold(fields[0], "from") {
			continue
		}
		for i := 1; i < len(fields); i++ {
			if strings.EqualFold(fields[i], "as") && i < len(fields)-1 {
				stages = append(stages, fields[i+1])
				break
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return stages, nil
}
