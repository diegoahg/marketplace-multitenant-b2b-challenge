import 'dart:async';
import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:marketplace_frontend/api.dart';
import 'package:marketplace_frontend/models.dart';
import 'helpers.dart';

void main() {
  test('new currencies preserve decimals, signs and unambiguous labels', () {
    for (final entry in {
      'COP': 'COP',
      'USD': 'USD',
      'GTQ': 'Q',
      'ARS': 'ARS',
    }.entries) {
      expect(
        Money(BigInt.from(123456), entry.key, 2).formatted,
        '${entry.value} 1.234,56',
      );
      expect(
        Money(BigInt.from(-1), entry.key, 2).formatted,
        '−${entry.value} 0,01',
      );
    }
  });
  for (final name in ['GT01', 'GT02', 'GT03', 'GT04', 'GT05']) {
    test('$name imported Go snapshot matches canonical backend golden', () {
      final q = goldenQuote(name);
      final expected = fixture(name);
      for (final field in {
        'grossSubtotal': 'gross',
        'discountTotal': 'discount',
        'taxableBase': 'base',
        'taxTotal': 'tax',
        'total': 'total',
      }.entries) {
        expect(
          q.money(field.key).amount.toString(),
          expected[field.value].toString(),
        );
      }
      expect(q.eligible, expected['eligible']);
      expect(
        q
            .rows('lines')
            .fold<int>(0, (sum, line) => sum + (line['comboQuantity'] as int)),
        expected['comboUnits'],
      );
      expect(
        q
            .rows('adjustments')
            .where((a) => a['promotionType'] == 'SCALE')
            .fold<int>(0, (sum, a) => sum + (a['quantity'] as int)),
        expected['scaleUnits'],
      );
      expect(
        q.rows('gifts').fold<int>(0, (sum, g) => sum + (g['quantity'] as int)),
        expected['giftUnits'],
      );
    });
  }
  test('GT09 exact decimal presentation beyond JavaScript safe integers', () {
    expect(
      Money(BigInt.parse('9007199254740993'), 'PEN', 2).formatted,
      'S/ 90.071.992.547.409,93',
    );
    expect(Money(BigInt.from(5), 'PEN', 3).formatted, 'S/ 0,005');
    expect(Money(BigInt.from(1005), 'PEN', 3).formatted, 'S/ 1,005');
    expect(Money(BigInt.from(10015), 'PEN', 3).formatted, 'S/ 10,015');
    expect(Money(BigInt.from(-101), 'PEN', 2).formatted, '−S/ 1,01');
    expect(Money(BigInt.from(12345), 'CLP', 0).formatted, '\$ 12.345');
  });
  test(
    'HTTP contract propagates scope, quantities and idempotency key',
    () async {
      final calls = <http.Request>[];
      final api = MarketplaceApi(
        baseUrl: 'http://test/api',
        client: MockClient((r) async {
          calls.add(r);
          return http.Response(
            jsonEncode(
              r.url.path.endsWith('quotes')
                  ? sampleQuote().json
                  : sampleOrder(sampleQuote()).json,
            ),
            201,
          );
        }),
      );
      final q = await api.quote(const Scope(), 'CREDIT', {
        'SKU-001': 12,
        'SKU-002': 0,
      });
      await api.confirm(q, 'test-key');
      expect(jsonDecode(calls.first.body)['items'], [
        {'sku': 'SKU-001', 'quantity': 12},
      ]);
      expect(jsonDecode(calls.first.body)['tenantId'], 'tenant-demo');
      expect(calls.first.headers['X-Country'], 'PE');
      expect(calls.last.headers['Idempotency-Key'], 'test-key');
      expect(jsonDecode(calls.last.body), {
        'quoteId': q.id,
        'customerId': 'CUSTOMER-001',
      });
      api.close();
    },
  );
  test('server errors preserve code and trace for recovery', () async {
    final api = MarketplaceApi(
      baseUrl: 'http://test',
      client: MockClient(
        (_) async => http.Response(
          '{"code":"CREDIT_INSUFFICIENT","message":"insufficient","traceId":"trace-1"}',
          409,
        ),
      ),
    );
    await expectLater(
      api.confirm(sampleQuote(), 'key'),
      throwsA(
        isA<ApiFailure>()
            .having((e) => e.code, 'code', 'CREDIT_INSUFFICIENT')
            .having((e) => e.traceId, 'trace', 'trace-1')
            .having((e) => e.definitive, 'definitive', true),
      ),
    );
    api.close();
  });
  test('GT05 ineligible quote cannot be confirmed', () async {
    final api = FakeApi()..result = goldenQuote('GT05');
    final c = controller(api: api);
    await c.initialize();
    c.preset({'SKU-001': 2});
    await c.createQuote();
    expect(c.canConfirm, false);
    await c.confirm();
    expect(api.keys, isEmpty);
    c.setPayment('CASH');
    expect(c.quote, isNull);
  });
  test(
    'GT06 confirmed total is the quote snapshot without recalculating',
    () async {
      final api = FakeApi();
      final c = controller(api: api);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      final quoted = c.quote!.total;
      await c.confirm();
      expect(c.confirmed!.total, quoted);
      expect(api.quoteCalls, 1);
      expect(c.orders, hasLength(1));
      expect(c.canConfirm, false);
    },
  );
  test(
    'GT06 reject a mismatching server snapshot without marking success',
    () async {
      final api = FakeApi()
        ..onConfirm = (q, _) async {
          final result = sampleOrder(q).json;
          result['total'] = {'amount': '9999', 'currency': 'PEN', 'scale': 2};
          return Order(result);
        };
      final c = controller(api: api);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      expect(c.failure!.code, 'SNAPSHOT');
      expect(c.confirmed, isNull);
      expect(c.pendingKey, isNotNull);
    },
  );
  test(
    'GT07 one in-flight request for 100 concurrent UI confirmations',
    () async {
      final result = Completer<Order>();
      final api = FakeApi()..onConfirm = (_, _) => result.future;
      final c = controller(api: api);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      final attempts = List.generate(100, (_) => c.confirm());
      await Future<void>.delayed(Duration.zero);
      expect(api.keys, hasLength(1));
      result.complete(sampleOrder(c.quote!));
      await Future.wait(attempts);
      expect(c.orders, hasLength(1));
    },
  );
  test(
    'GT07 uncertain response survives reload and reuses key even after expiry',
    () async {
      final storage = MemoryStorage();
      final api = FakeApi()
        ..confirmFailure = const ApiFailure('TIMEOUT', 'timeout');
      final c = controller(api: api, storage: storage);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      final key = c.pendingKey;
      expect(key, isNotNull);
      c.setCountry('CL');
      c.quantity('SKU-001', 3);
      expect(c.scope.country, 'PE');
      expect(c.items['SKU-001'], 12);
      final restored = controller(
        api: api,
        storage: storage,
        now: () => fixedNow.add(const Duration(hours: 1)),
      );
      await restored.initialize();
      expect(restored.pendingKey, key);
      expect(restored.canConfirm, true);
      api.confirmFailure = null;
      await restored.confirm();
      expect(api.keys, [key, key]);
      expect(restored.orders, hasLength(1));
      expect(storage.saved!['pending'], isNull);
    },
  );
  test(
    'GT08 changed credit rejects order and requires a fresh quote',
    () async {
      final api = FakeApi()
        ..confirmFailure = const ApiFailure(
          'CREDIT_INSUFFICIENT',
          'changed credit',
          status: 409,
        );
      final c = controller(api: api);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      expect(c.orders, isEmpty);
      expect(c.quote, isNull);
      expect(c.pendingKey, isNull);
      expect(c.failure!.code, 'CREDIT_INSUFFICIENT');
    },
  );
  test('GT10 country change invalidates quote and scopes history', () async {
    final api = FakeApi();
    final c = controller(api: api);
    await c.initialize();
    c.preset({'SKU-001': 12});
    await c.createQuote();
    await c.confirm();
    c.setCountry('CL');
    expect(c.quote, isNull);
    expect(c.scopedOrders, isEmpty);
    c.setCountry('PE');
    expect(c.scopedOrders, hasLength(1));
  });
  test(
    'GT11 server failure keeps intent and never invents a confirmed order',
    () async {
      final api = FakeApi()
        ..confirmFailure = const ApiFailure(
          'INTERNAL_ERROR',
          'error',
          status: 500,
        );
      final c = controller(api: api);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      expect(c.orders, isEmpty);
      expect(c.pendingKey, isNotNull);
      expect(c.confirmed, isNull);
    },
  );
  test(
    'GT12 replay after storage failure retains just one local order',
    () async {
      final storage = MemoryStorage();
      final api = FakeApi()
        ..onConfirm = (q, _) async {
          storage.failWrites = true;
          return sampleOrder(q);
        };
      final c = controller(api: api, storage: storage);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      final key = c.pendingKey;
      expect(c.confirmed, isNull);
      expect(c.orders, hasLength(1));
      storage.failWrites = false;
      api.onConfirm = null;
      await c.confirm();
      expect(api.keys, [key, key]);
      expect(c.orders, hasLength(1));
      expect(c.pendingKey, isNull);
    },
  );
  test(
    'storage failure before sending never submits an unsafe confirmation',
    () async {
      final storage = MemoryStorage()..failWrites = true;
      final api = FakeApi();
      final c = controller(api: api, storage: storage);
      await c.initialize();
      c.preset({'SKU-001': 12});
      await c.createQuote();
      await c.confirm();
      expect(api.keys, isEmpty);
      expect(c.failure!.code, 'STORAGE');
    },
  );
  test('expired quote cannot start a new confirmation', () async {
    final c = controller(now: () => fixedNow.add(const Duration(minutes: 15)));
    await c.initialize();
    c.preset({'SKU-001': 12});
    await c.createQuote();
    expect(c.canConfirm, false);
  });
  test(
    'editing cart or payment invalidates quote; empty cart is not sent',
    () async {
      final api = FakeApi();
      final c = controller(api: api);
      await c.initialize();
      await c.createQuote();
      expect(api.quoteCalls, 0);
      c.preset({'SKU-001': 12});
      await c.createQuote();
      c.quantity('SKU-001', 11);
      expect(c.quote, isNull);
      await c.createQuote();
      c.setPayment('CASH');
      expect(c.quote, isNull);
      c.quantity('SKU-001', 0);
      expect(c.items, isEmpty);
    },
  );
}
