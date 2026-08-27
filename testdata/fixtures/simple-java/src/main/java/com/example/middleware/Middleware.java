package com.example.middleware;

import com.example.auth.Token;

public class Middleware {
    public static String authenticate(String request) {
        return Token.sign(request);
    }
}
