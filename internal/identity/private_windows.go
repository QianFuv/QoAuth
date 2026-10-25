package identity

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"sort"
	"strings"
)

func privateDescriptor(directory bool) (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	flags := ""
	if directory {
		flags = "OICI"
	}
	return windows.SecurityDescriptorFromString("D:P(A;" + flags + ";FA;;;" + user.User.Sid.String() + ")(A;" + flags + ";FA;;;SY)(A;" + flags + ";FA;;;BA)")
}
func restrictFile(path string) error {
	return restrictPath(path, false)
}
func restrictDirectory(path string) error { return restrictPath(path, true) }
func restrictPath(path string, directory bool) error {
	descriptor, err := privateDescriptor(directory)
	if err != nil {
		return err
	}
	acl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
func checkPrivateFile(path string, _ os.FileInfo) error {
	return checkPrivatePath(path, false)
}
func checkPrivateDirectory(path string, _ os.FileInfo) error { return checkPrivatePath(path, true) }
func checkPrivatePath(path string, directory bool) error {
	actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	wanted, err := privateDescriptor(directory)
	if err != nil {
		return err
	}
	if normalizedDacl(actual.String()) != normalizedDacl(wanted.String()) {
		return errors.New("key file access must be restricted to its owner, SYSTEM and administrators")
	}
	return nil
}

func normalizedDacl(value string) string {
	parts := strings.Split(value, "(")
	if len(parts) != 4 || (parts[0] != "D:P" && parts[0] != "D:PAI") {
		return value
	}
	sort.Strings(parts[1:])
	return "D:P(" + strings.Join(parts[1:], "(")
}
