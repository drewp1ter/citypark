#!/usr/bin/env python3
"""Contract checks against an owner-confirmed isolated 1C publication (stdlib only).

Default: advisory availability and unknown-key lookup, without reservation writes.
Creation requires --create --isolated-dev and leaves one reservation for inspection.
Credentials are prompted; never written to disk or included in diagnostics.
"""
import argparse
import base64
import copy
import datetime as dt
import getpass
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
import uuid


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class CheckFailure(Exception):
    pass


def require(condition, message):
    if not condition:
        raise CheckFailure(message)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', required=True, help='Service root ending /hs/booking/v1')
    parser.add_argument('--room-type', default='single-balcony')
    parser.add_argument('--check-in', required=True, type=dt.date.fromisoformat)
    parser.add_argument('--check-out', required=True, type=dt.date.fromisoformat)
    parser.add_argument('--guests', type=int, default=1)
    parser.add_argument('--create', action='store_true')
    parser.add_argument('--isolated-dev', action='store_true', help='Owner confirms publication is an isolated development base')
    parser.add_argument('--nightly-price-kopecks', type=int)
    args = parser.parse_args()
    parts = urllib.parse.urlsplit(args.base_url)
    require(parts.scheme in ('http', 'https') and parts.hostname and not parts.username and not parts.password,
            'Provide an HTTP(S) URL without embedded credentials.')
    require(not parts.query and not parts.fragment and parts.path.rstrip('/').endswith('/hs/booking/v1'),
            'Expected service root ending /hs/booking/v1, without query or fragment.')
    require(args.check_out > args.check_in, 'Checkout must be after check-in.')
    if args.create:
        require(args.isolated_dev, 'Creation requires --isolated-dev; never use a production publication.')
        require(args.nightly_price_kopecks and args.nightly_price_kopecks > 0,
                'Creation requires a positive --nightly-price-kopecks.')
    username = os.environ.get('ONEC_BOOKING_USER') or input('Development service username: ')
    password = os.environ.get('ONEC_BOOKING_PASSWORD') or getpass.getpass('Development service password: ')
    auth = 'Basic ' + base64.b64encode((username + ':' + password).encode()).decode('ascii')
    opener = urllib.request.build_opener(NoRedirect())
    root = args.base_url.rstrip('/')

    def request(method, path, body=None, authorized=True):
        payload = None if body is None else json.dumps(body, ensure_ascii=False).encode('utf-8')
        headers = {'Content-Type': 'application/json; charset=utf-8'}
        if authorized:
            headers['Authorization'] = auth
        req = urllib.request.Request(root + path, data=payload, headers=headers, method=method)
        try:
            response = opener.open(req, timeout=15)
        except urllib.error.HTTPError as exc:
            response = exc
        except (OSError, TimeoutError, urllib.error.URLError):
            raise CheckFailure('Transport failure; if creation was dispatched, recover using the printed key.') from None
        with response:
            status = response.code
            data = response.read(65537)
            require(len(data) <= 65536, 'Response exceeded 64 KiB.')
            if status in (401, 403) and not authorized:
                return status, None
            require(response.headers.get('Cache-Control') == 'no-store', 'Missing Cache-Control: no-store.')
            require(response.headers.get_content_type() == 'application/json', 'Expected JSON response.')
            try:
                decoded = json.loads(data.decode('utf-8-sig'))
            except (UnicodeError, ValueError):
                raise CheckFailure('Response was not UTF-8 JSON; body omitted.') from None
            return status, decoded

    def error(result, status, code, message):
        actual_status, body = result
        require(actual_status == status, f'Expected status {status}; received {actual_status}. Body omitted.')
        require(body == {'statusCode': status, 'code': code, 'message': message}, f'Unexpected {code} error envelope.')

    stay = {'roomTypeId': args.room_type, 'checkIn': str(args.check_in),
            'checkOut': str(args.check_out), 'guests': args.guests}
    status, _ = request('POST', '/availability', stay, authorized=False)
    require(status in (401, 403), 'Anonymous service access was not denied.')
    status, availability = request('POST', '/availability', stay)
    require(status == 200, f'Availability returned HTTP {status}; body omitted.')
    require(set(availability) == set(stay) | {'available'}, 'Unexpected availability keys.')
    require(all(availability[k] == v for k, v in stay.items()), 'Availability changed stay parameters.')
    require(type(availability['available']) is bool, 'Availability flag must be boolean.')
    error(request('GET', '/reservations/' + str(uuid.uuid4())), 404,
          'RESERVATION_NOT_FOUND', 'Бронирование не найдено')
    invalid = dict(stay, extra='unknown')
    error(request('POST', '/availability', invalid), 400,
          'INVALID_REQUEST', 'Проверьте данные бронирования')
    print('PASS: anonymous access, availability, unknown-key lookup, unknown-field rejection')
    if not args.create:
        print('No reservation creation requested. Designer/register/concurrency checks remain separate.')
        return
    require(availability['available'], 'Selected stay is unavailable; no creation attempted.')
    key = str(uuid.uuid4())
    print('Recovery key (retain this even after timeout): ' + key, flush=True)
    nights = (args.check_out - args.check_in).days
    create = dict(stay, idempotencyKey=key,
                  guest={'name': 'Тест онлайн-бронирования', 'email': 'booking-check@example.invalid',
                         'phone': '+70000000000'},
                  price={'currency': 'RUB', 'nights': nights,
                         'nightlyPriceKopecks': args.nightly_price_kopecks,
                         'totalKopecks': args.nightly_price_kopecks * nights})
    status, saved = request('POST', '/reservations', create)
    require(status == 201, f'First creation returned HTTP {status}; recover by the printed key before further writes.')
    require(set(saved) == {'reservationId', 'status', 'price'} | set(stay), 'Unexpected reservation keys.')
    require(saved['status'] == 'confirmed', 'Reservation was not confirmed.')
    require(str(uuid.UUID(saved['reservationId'])) == saved['reservationId'], 'Noncanonical reservation UUID.')
    require(saved['price'] == create['price'] and all(saved[k] == v for k, v in stay.items()),
            'Reservation changed the accepted stay or price.')
    print('Created reservation: ' + saved['reservationId'], flush=True)
    replay = copy.deepcopy(create)
    replay['guest']['name'] = ' ' + replay['guest']['name'] + ' '
    replay['guest']['email'] = 'booking-check@EXAMPLE.INVALID'
    replay['guest']['phone'] = '+7 (000) 000-00-00'
    # Reverse field order and normalize contacts: identity must use values, not raw JSON.
    replay = dict(reversed(list(replay.items())))
    require(request('POST', '/reservations', replay) == (200, saved), 'Same-payload replay did not return the saved result.')
    require(request('GET', '/reservations/' + key) == (200, saved), 'Lookup did not return the saved result.')
    changed = copy.deepcopy(create)
    changed['guest']['name'] = 'Другой тестовый гость'
    error(request('POST', '/reservations', changed), 409,
          'IDEMPOTENCY_CONFLICT', 'Ключ бронирования уже использован с другими данными')
    print('PASS: create, normalized/reordered replay, recovery, changed-payload conflict')
    print('One reservation remains for document/register inspection. Cancel it using the staff workflow.')
    print('Concurrency, rollback, adjacent stays and staff check-in pricing are NOT verified by this script.')


if __name__ == '__main__':
    try:
        main()
    except (CheckFailure, ValueError) as exc:
        print('FAIL: ' + str(exc), file=sys.stderr)
        sys.exit(1)
