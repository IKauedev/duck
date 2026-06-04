package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func Confirm(message string) (bool, error) {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "s" || answer == "sim" || answer == "y" || answer == "yes", nil
}
