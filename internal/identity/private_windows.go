package identity

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"sort"
	"strings"
)

func privateDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)")
}
func restrictFile(path string) error {
	descriptor, err := privateDescriptor()
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
	actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	wanted, err := privateDescriptor()
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
