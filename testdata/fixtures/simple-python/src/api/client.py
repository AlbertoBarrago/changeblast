from ..auth import middleware


def handle(request):
    return middleware.authenticate(request)
