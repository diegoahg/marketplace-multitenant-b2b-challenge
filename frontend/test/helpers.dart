import 'dart:convert';
import 'dart:io';
import 'package:marketplace_frontend/api.dart';
import 'package:marketplace_frontend/models.dart';
import 'package:marketplace_frontend/shop_controller.dart';
import 'package:marketplace_frontend/store.dart';

final fixedNow = DateTime.utc(2026, 1, 1, 12);
Json fixture(String name) =>
    jsonDecode(File('test/fixtures/$name.json').readAsStringSync()) as Json;
Quote sampleQuote() => Quote(fixture('quote'));
Order sampleOrder(Quote q) => Order({
  ...q.scope.toJson(),
  'orderId': '22222222-2222-4222-8222-222222222222',
  'orderNumber': 'ORD-22222222',
  'quoteId': q.id,
  'currency': q.total.currency,
  'total': q.json['total'],
  'status': 'CONFIRMED',
  'createdAt': fixedNow.toIso8601String(),
});

class MemoryStorage implements CheckoutStorage {
  Json? saved;
  bool failWrites = false;
  @override
  Future<Json?> read() async => saved;
  @override
  Future<void> write(Json state) async {
    if (failWrites) throw StateError('storage unavailable');
    saved = jsonDecode(jsonEncode(state));
  }
}

class FakeApi extends MarketplaceApi {
  Quote result = sampleQuote();
  ApiFailure? confirmFailure;
  final keys = <String>[];
  int quoteCalls = 0;
  Future<Order> Function(Quote, String)? onConfirm;
  Scope? searchedScope;
  @override
  Future<bool> ready() async => true;
  @override
  Future<Quote> quote(
    Scope scope,
    String payment,
    Map<String, int> items,
  ) async {
    quoteCalls++;
    return result;
  }

  @override
  Future<Order> confirm(Quote q, String key) async {
    keys.add(key);
    if (onConfirm != null) return onConfirm!(q, key);
    if (confirmFailure != null) throw confirmFailure!;
    return sampleOrder(q);
  }

  @override
  Future<Order> order(Scope scope, String id) async {
    searchedScope = scope;
    return sampleOrder(result);
  }
}

ShopController controller({
  FakeApi? api,
  MemoryStorage? storage,
  DateTime Function()? now,
}) => ShopController(
  api ?? FakeApi(),
  storage ?? MemoryStorage(),
  now: now ?? () => fixedNow,
);

/// Serialized by the actual Go engine using the backend golden test scenarios.
Quote goldenQuote(String name) => Quote(fixture('quote_$name'));
