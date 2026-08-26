from .token import sign_token


def authenticate(request):
    return sign_token(request)
