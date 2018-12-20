import asyncio
import datetime
import logging
import os

from sanic import Sanic
from sanic.exceptions import SanicException
from sanic.response import json

STARTED = datetime.datetime.utcnow().isoformat()
DIR = os.path.dirname(os.path.realpath(__file__))

app = Sanic()
app.config.REQUEST_MAX_SIZE = 100000  # 100 KB
app.config.REQUEST_TIMEOUT = 5  # 5 sec
app.config.RESPONSE_TIMEOUT = 15  # 15 sec
app.config.KEEP_ALIVE = False  # Drop connection after resp


@app.exception(SanicException)
def json_error(request, exception):
    e = exception
    if request:
        logging.error(f'Error {e.status_code}, {request.url}: {str(e)}')

    return json({'status': e.status_code, 'message': str(e)}, e.status_code)


@app.route('/')
async def root(req):
    return json({
        'hello': 'world'
    })


SHUTTING_DOWN = False    

@app.listener('before_server_stop')
async def notify_server_stopping(app, loop):
    global SHUTTING_DOWN
    if not os.environ.get('UNGRACEFUL'):
        print('Sleeping before shutting down!')
        SHUTTING_DOWN = True    
        await asyncio.sleep(15)
        print('Server shutting down!')
    else:
        print('Quit immediately')

@app.route('/status')
async def health_check(req):
    status = 200
    if SHUTTING_DOWN:
        status = 500
    return json({
        'service': 'graceful-py',
        'startedAt': STARTED,
        'status': status
    },status=status)


def main():
    app.static('/favicon.ico', DIR + '/favicon.ico')
    app.run(host='0.0.0.0', port=8080, debug=True)


if __name__ == '__main__':
    main()
