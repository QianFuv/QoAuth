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
	if err := rejectReparse(path); err != nil {
		return err
	}
	actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := actual.Owner()
	if err != nil || !trustedOwner(owner) {
		return errors.New("storage owner is not trusted")
	}
	wanted, err := privateDescriptor(directory)
	if err != nil {
		return err
	}
	value := actual.String()
	position := strings.Index(value, "D:")
	if position < 0 {
		return errors.New("private access list required")
	}
	value = value[position:]
	if normalizedDacl(value) != normalizedDacl(wanted.String()) {
		return errors.New("key file access must be restricted to its owner, SYSTEM and administrators")
	}
	return nil
}

func trustedOwner(owner *windows.SID) bool {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	return err == nil && owner != nil && (owner.String() == user.User.Sid.String() || owner.IsWellKnown(windows.WinLocalSystemSid) || owner.IsWellKnown(windows.WinBuiltinAdministratorsSid))
}

func checkDatabaseFile(path string, _ os.FileInfo) error {
	if err := rejectReparse(path); err != nil {
		return err
	}
	actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := actual.Owner()
	if err != nil || !trustedOwner(owner) {
		return errors.New("storage owner is not trusted")
	}
	wanted, err := privateDescriptor(false)
	if err != nil {
		return err
	}
	value := actual.String()
	position := strings.Index(value, "D:")
	if position < 0 {
		return errors.New("database access list required")
	}
	value = value[position:]
	value = strings.ReplaceAll(value, ";ID;", ";;")
	value = strings.ReplaceAll(value, "D:AI", "D:P")
	if normalizedDacl(value) != normalizedDacl(wanted.String()) {
		return errors.New("database access must be private")
	}
	return nil
}

func rejectReparse(path string) error {
	encoded, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes, err := windows.GetFileAttributes(encoded)
	if err != nil {
		return err
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return errors.New("storage reparse points are not allowed")
	}
	return nil
}

func checkReparse(path string) error { return rejectReparse(path) }

func singleLink(file *os.File, _ os.FileInfo) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return err
	}
	if info.NumberOfLinks != 1 {
		return errors.New("hard-link aliases are not allowed")
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
