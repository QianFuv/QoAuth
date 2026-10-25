/** Behavior tests for client-side form helpers that mirror server rules. */
import { test } from "node:test";
import assert from "node:assert/strict";
import { describeError, isPasswordWithinPolicy, splitCommaList, splitLines } from "../src/validation.ts";

const limits = { min_characters: 15, max_bytes: 1024 };

test("password length counts NFC code points, not UTF-16 units", () => {
  assert.equal(isPasswordWithinPolicy("😀".repeat(15), limits), true);
  assert.equal(isPasswordWithinPolicy("😀".repeat(14), limits), false);
  assert.equal(isPasswordWithinPolicy("é".repeat(15), limits), true);
  assert.equal(isPasswordWithinPolicy("é".repeat(7) + "x", limits), false);
});

test("password byte limit applies to UTF-8 bytes and keeps spaces", () => {
  assert.equal(isPasswordWithinPolicy(" ".repeat(15), limits), true);
  assert.equal(isPasswordWithinPolicy("a".repeat(1024), limits), true);
  assert.equal(isPasswordWithinPolicy("a".repeat(1025), limits), false);
  assert.equal(isPasswordWithinPolicy("é".repeat(513), limits), false);
});

test("callback URLs split only at line breaks so commas survive", () => {
  assert.deepEqual(splitLines("https://a.test/cb?next=one,two\r\n\n https://b.test/cb \n"), ["https://a.test/cb?next=one,two", "https://b.test/cb"]);
});

test("scope lists split at commas and drop empty entries", () => {
  assert.deepEqual(splitCommaList(" openid, profile,,notes.read "), ["openid", "profile", "notes.read"]);
  assert.deepEqual(splitCommaList(null), []);
});

test("server errors become Chinese sentences without inventing detail", () => {
  assert.equal(describeError("invalid credentials\n"), "用户名或密码不正确。");
  assert.equal(describeError("password policy: choose a less common password"), "密码过于常见，请换一个。");
  assert.equal(describeError("password policy: at least 15 characters after Unicode normalization and at most 1024 UTF-8 bytes"), "密码长度不符合要求：Unicode 规范化后的字符数或 UTF-8 字节数超出限制。");
  assert.equal(describeError("something unexpected"), "请求未成功：something unexpected");
  assert.equal(describeError(""), "请求未成功，请重试。");
});
