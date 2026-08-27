package com.example.auth;

public class Token {
    public static String sign(String payload) {
        return "signed:" + payload;
    }
}
