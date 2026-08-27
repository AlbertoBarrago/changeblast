package com.example.api;

import com.example.middleware.Middleware;

public class Client {
    public static String handle(String request) {
        return Middleware.authenticate(request);
    }
}
