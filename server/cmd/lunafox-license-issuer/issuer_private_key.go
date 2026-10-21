package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const privateKeyPrefix = "LF-ISSUER-ED25519-1:"

func loadOrCreatePrivateKey(path string) (ed25519.PrivateKey, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("创建私钥目录: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		_, privateKey, generateErr := ed25519.GenerateKey(nil)
		if generateErr != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return nil, generateErr
		}
		encoded := privateKeyPrefix + base64.RawURLEncoding.EncodeToString(privateKey.Seed()) + "\n"
		_, writeErr := io.WriteString(file, encoded)
		if writeErr == nil {
			writeErr = file.Sync()
		}
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.Remove(path)
			return nil, fmt.Errorf("保存私钥: %w", writeErr)
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return nil, fmt.Errorf("关闭私钥: %w", closeErr)
		}
		return privateKey, nil
	}
	if !os.IsExist(err) {
		return nil, fmt.Errorf("创建私钥文件: %w", err)
	}
	return loadPrivateKey(path)
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("未找到作者私钥，请从备份恢复；首次创建发行身份须使用 --export-public-key 并构建匹配产品")
	}
	if err != nil {
		return nil, fmt.Errorf("检查私钥: %w", err)
	}
	// Never replace unreadable or malformed material with a fresh identity: doing
	// so would silently orphan every previously issued customer key.
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() < 1 || info.Size() > 256 {
		return nil, fmt.Errorf("私钥必须为仅当前用户可读写的普通文件（权限0600）")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取私钥: %w", err)
	}
	encoded := strings.TrimSuffix(string(content), "\n")
	if !strings.HasPrefix(encoded, privateKeyPrefix) {
		return nil, fmt.Errorf("私钥格式无效；未更改原文件")
	}
	seed, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(encoded, privateKeyPrefix))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("私钥格式无效；未更改原文件")
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
