package main

import (
	"os"
	"testing"
)

func TestReadRecoveryCredentialFromFileDescriptor(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if _, err = writer.WriteString("protected-secret\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	credential, err := readRecoveryCredential(int(reader.Fd()))
	if err != nil || credential != "protected-secret" {
		t.Fatalf("credential=%q err=%v", credential, err)
	}
}
