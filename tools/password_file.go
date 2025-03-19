package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var password string
	var filename string
	var show bool

	// Parse command-line arguments
	flag.StringVar(&password, "password", "", "Password to save to file")
	flag.StringVar(&filename, "file", "mysql_password.txt", "Output filename")
	flag.BoolVar(&show, "show", false, "Display commands to use the password file")
	flag.Parse()

	if password == "" {
		fmt.Println("Error: Password is required")
		fmt.Println("Usage: password_file -password \"your_complex_password\" [-file filename.txt] [-show]")
		os.Exit(1)
	}

	// Create absolute path in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Create file in home directory if path not absolute
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(homeDir, filename)
	}

	// Write password to file
	if err := os.WriteFile(filename, []byte(password), 0600); err != nil {
		fmt.Printf("Error writing password file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password saved to %s with secure permissions (0600)\n", filename)

	if show {
		fmt.Println("\nTo use this password file with go-create:")
		fmt.Printf("  ./bin/go-create -create-user USERNAME -create-pass \"$(< %s)\" ...\n", filename)
		fmt.Println("\nTo test connection with this password:")
		fmt.Printf("  ./bin/go-create -test-connection -user USERNAME -pass \"$(< %s)\" -host HOSTNAME\n", filename)
		fmt.Println("\nFor MySQL client:")
		fmt.Printf("  mysql -u USERNAME -p\"$(< %s)\" -h HOST\n", filename)
		fmt.Println("\nTo delete this file when done:")
		fmt.Printf("  rm %s\n", filename)
	}
}
