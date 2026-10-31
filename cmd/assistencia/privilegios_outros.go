//go:build !linux

package main

func reduzirPrivilegios(string) error { return nil }
