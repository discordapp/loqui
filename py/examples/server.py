from gevent.monkey import patch_all
patch_all()

from drpc.server import DRPCServer


class Server(DRPCServer):
    def handle_request(self, request):
        return 'OK!'


if __name__ == '__main__':
    s = Server(('localhost', 4001))
    s.serve_forever()
