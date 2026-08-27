#include "../auth/middleware.h"

const char *handle(const char *request) {
    return authenticate(request);
}
