package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for yes/no confirmation
// Returns true if user confirms, false otherwise
func Confirm(message string) bool {
	fmt.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ConfirmDeletion prompts for confirmation of a destructive operation
// Shows resource details and requires explicit confirmation
func ConfirmDeletion(resourceType, name, id string) bool {
	fmt.Printf("\nYou are about to delete the following %s:\n", resourceType)
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("  ID:   %s\n", id)
	fmt.Println()

	return Confirm("Are you sure you want to delete this resource?")
}

// ConfirmDataDeletion prompts for confirmation of an irreversible data operation
// Requires the user to type the resource name exactly to confirm
// Returns true if confirmed, false otherwise
func ConfirmDataDeletion(resourceType, name string) bool {
	fmt.Printf("\n⚠️  WARNING: This operation is IRREVERSIBLE and will delete all data\n")
	fmt.Printf("  Resource Type: %s\n", resourceType)
	fmt.Printf("  Name:          %s\n", name)
	fmt.Println()
	fmt.Printf("Type the %s name '%s' to confirm: ", resourceType, name)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(response)
	return response == name
}

// ValidateConfirmFlag checks if the --confirm flag value matches the resource name
// Used for non-interactive confirmation of data deletion
func ValidateConfirmFlag(confirmValue, resourceName string) bool {
	return confirmValue == resourceName
}
