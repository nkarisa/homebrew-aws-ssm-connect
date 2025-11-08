package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Instance represents the structure of the data returned by the JMESPath query.
type Instance struct {
	InstanceID       string `json:"InstanceId"`
	Name             string `json:"Name"`
	PrivateIPAddress string `json:"PrivateIpAddress"`
}

// This program executes the AWS CLI command to list EC2 instances, parses the
// results, and allows the user to select an instance for detail viewing or SSM session.
func main() {
	fmt.Println("--- AWS EC2 Instance Lister (Interactive Selection) ---")

	// The JMESPath query is used to flatten the Reservations and Instances arrays
	// and select the required fields. The output must be JSON for programmatic parsing.
	const instanceQuery = "Reservations[*].Instances[*].{InstanceId:InstanceId,Name:Tags[?Key==`Name`].Value | [0],PrivateIpAddress:PrivateIpAddress}"

	args := []string{
		"ec2",
		"describe-instances",
		"--query", instanceQuery,
		"--output", "json", // Output is JSON for programmatic parsing
	}

	// Check if the user provided an AWS Profile argument
	profile := getProfileFromArgs()
	if profile != "" {
		fmt.Printf("Using AWS Profile: %s\n", profile)
		args = append(args, "--profile", profile)
	} else {
		fmt.Println("No profile specified. Using the default profile/active environment.")
	}

	// 1. Execute the command and capture output
	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf("Error executing AWS CLI command: %v\n", err)
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "AWS CLI Error Output:\n%s\n", exitError.Stderr)
		}
		fmt.Println("\nPossible issues:")
		fmt.Println("1. Is the 'aws' CLI installed and in your PATH?")
		fmt.Println("2. Is the specified profile configured for SSO and active (run 'aws sso login')?")
		fmt.Println("3. Do you have the necessary EC2 permissions and SSM Agent running on the instances?")
		os.Exit(1)
	}

	// 2. Parse and flatten the JSON output (handling array-of-arrays structure)
	var rawReservations [][]Instance
	if err := json.Unmarshal(output, &rawReservations); err != nil {
		fmt.Printf("Error parsing JSON output from AWS CLI: %v\n", err)
		os.Exit(1)
	}

	// Flatten the raw array of arrays into a single slice of Instance
	var instances []Instance
	for _, reservationInstances := range rawReservations {
		instances = append(instances, reservationInstances...)
	}

	if len(instances) == 0 {
		fmt.Println("\nNo EC2 instances found.")
		return
	}

	// 3. Prompt user for selection
	selectedID, err := promptForSelection(instances)
	if err != nil {
		if err.Error() == "quit signal" {
			fmt.Println("\nExiting program.")
			os.Exit(0) // Graceful exit on 'q'
		}
		fmt.Printf("\nSelection Error: %v\n", err)
		os.Exit(1)
	}

	// 4. Start the SSM Session to the selected instance
	// We no longer need to find the full instance object, just the ID and profile.
	startSSMSession(selectedID, profile)
}

// getProfileFromArgs extracts the --profile argument from command line arguments.
func getProfileFromArgs() string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--profile" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// promptForSelection lists instances with numbered options and asks the user to input the option number.
func promptForSelection(instances []Instance) (string, error) {
	fmt.Println("\nAvailable EC2 Instances:")
	fmt.Println("-----------------------------------------------------------------------------------------")
	// Header formatting: 8 chars for Option, 20 for ID, 30 for Name, 15 for IP
	fmt.Printf("%-8s %-20s %-30s %-15s\n", "OPTION", "INSTANCE ID", "NAME", "PRIVATE IP")
	fmt.Println("-----------------------------------------------------------------------------------------")

	for i, inst := range instances {
		name := inst.Name
		if name == "" {
			name = "N/A"
		}
		// Print the 1-based index (i+1) as the option number
		fmt.Printf("%-8d %-20s %-30s %-15s\n", i+1, inst.InstanceID, name, inst.PrivateIPAddress)
	}
	fmt.Println("-----------------------------------------------------------------------------------------")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	// Updated prompt to include the quit option
	fmt.Print("Enter the option number to start an SSM Session (or 'q' to quit): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	trimmedInput := strings.ToLower(strings.TrimSpace(input))

	// Check for quit signal
	if trimmedInput == "q" {
		return "", fmt.Errorf("quit signal")
	}

	selectedNum, err := strconv.Atoi(trimmedInput)
	if err != nil {
		return "", fmt.Errorf("invalid input: '%s' is not a valid number or 'q'", trimmedInput)
	}

	// Validate the selected number is within bounds (1 to length)
	if selectedNum < 1 || selectedNum > len(instances) {
		return "", fmt.Errorf("invalid option number: %d. Must be between 1 and %d", selectedNum, len(instances))
	}

	// Get the InstanceID using the 0-based index (selectedNum - 1)
	return instances[selectedNum-1].InstanceID, nil
}

// startSSMSession executes 'aws ssm start-session' with the selected Instance ID.
func startSSMSession(instanceID string, profile string) {
	fmt.Printf("\nAttempting to start SSM session for Instance ID: %s...\n", instanceID)

	args := []string{
		"ssm",
		"start-session",
		"--target", instanceID,
	}

	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.Command("aws", args...)

	// Crucial: Connect the command's I/O to the current process's I/O
	// This allows the user to interact with the SSM session directly.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and wait for it to complete
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nError starting SSM session: %v\n", err)
		fmt.Println("\nCheck if:")
		fmt.Println("1. The SSM Plugin is installed for the AWS CLI.")
		fmt.Println("2. The instance is running and the SSM Agent is healthy.")
		fmt.Println("3. The instance's IAM role has the necessary SSM permissions (e.g., AmazonSSMManagedInstanceCore).")
		// The exit code of the SSM session is propagated
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("SSM session terminated with exit code: %d\n", exitError.ExitCode())
		}
	} else {
		fmt.Println("\nSSM Session terminated successfully.")
	}
}
