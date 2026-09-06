import 'package:flutter/foundation.dart';
import 'api.dart';
import 'models.dart';
import 'store.dart';

class ShopController extends ChangeNotifier {
  final MarketplaceApi api;
  final CheckoutStorage storage;
  final DateTime Function() now;
  final List<Product> catalog;
  Scope scope = const Scope();
  String payment = 'CREDIT';
  Map<String, int> items = {};
  Quote? quote;
  Order? confirmed;
  List<Order> orders = [];
  String? pendingKey;
  bool initialized = false, busy = false, checking = false;
  bool? online;
  ApiFailure? failure;
  String? notice;
  ShopController(
    this.api,
    this.storage, {
    DateTime Function()? now,
    this.catalog = products,
  }) : now = now ?? DateTime.now;
  String nameOf(String sku) {
    for (final product in catalog) {
      if (product.sku == sku) return product.name;
    }
    return productName(sku);
  }

  bool get locked => busy || pendingKey != null || !initialized;
  int get units => items.values.fold(0, (a, b) => a + b);
  bool get canConfirm =>
      !busy &&
      quote != null &&
      confirmed == null &&
      (pendingKey != null ||
          (!quote!.expired(now()) && (payment == 'CASH' || quote!.eligible)));
  List<Order> get scopedOrders =>
      orders.where((o) => o.scope.key == scope.key).toList();

  Future<void> initialize() async {
    try {
      final state = await storage.read();
      if (state != null) {
        orders = (state['orders'] as List? ?? [])
            .map((e) => Order(Map<String, dynamic>.from(e)))
            .toList();
        if (state['pending'] != null) {
          quote = Quote(Map<String, dynamic>.from(state['pending']['quote']));
          pendingKey = state['pending']['key'];
          scope = quote!.scope;
          payment = quote!.payment;
          items = {
            for (final line in quote!.rows('lines'))
              line['sku'] as String: line['quantity'] as int,
          };
          notice =
              'Tienes una confirmación pendiente. Reintenta para recuperar el resultado sin duplicar tu pedido.';
        }
      }
      initialized = true;
    } catch (_) {
      failure = const ApiFailure(
        'STORAGE',
        'No pudimos recuperar tu compra guardada. Habilita el almacenamiento del navegador y recarga la página.',
      );
    }
    notifyListeners();
    await checkHealth();
  }

  Future<void> checkHealth() async {
    if (checking) return;
    checking = true;
    notifyListeners();
    online = await api.ready();
    checking = false;
    notifyListeners();
  }

  void edit(void Function() change) {
    if (locked) return;
    change();
    quote = null;
    confirmed = null;
    failure = null;
    notice = null;
    notifyListeners();
  }

  void quantity(String sku, int value) => edit(() {
    if (value <= 0) {
      items.remove(sku);
    } else {
      items[sku] = value.clamp(1, 1000000);
    }
  });
  void setCountry(String country) => edit(() {
    scope = Scope(country: country);
  });
  void setPayment(String value) => edit(() {
    payment = value;
  });
  void preset(Map<String, int> cart) => edit(() {
    items = Map.of(cart);
  });
  Future<void> createQuote() async {
    if (locked || items.isEmpty) return;
    busy = true;
    failure = null;
    notice = null;
    confirmed = null;
    quote = null;
    notifyListeners();
    try {
      quote = await api.quote(scope, payment, Map.of(items));
    } on ApiFailure catch (e) {
      failure = e;
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Json state({bool clearPending = false}) => {
    'orders': orders.take(50).map((o) => o.json).toList(),
    if (!clearPending && pendingKey != null && quote != null)
      'pending': {'key': pendingKey, 'quote': quote!.json},
  };
  Future<void> confirm() async {
    if (!canConfirm) return;
    busy = true;
    failure = null;
    notice = null;
    pendingKey ??= newKey();
    notifyListeners();
    try {
      // Persist BEFORE sending. Ambiguous errors keep this key, including across reloads.
      await storage.write(state());
      final result = await api.confirm(quote!, pendingKey!);
      if (result.quoteId != quote!.id || result.total != quote!.total) {
        throw const ApiFailure(
          'SNAPSHOT',
          'El pedido no coincide con la cotización. Conservamos tu intento para verificarlo.',
        );
      }
      orders = [result, ...orders.where((o) => o.id != result.id)];
      await storage.write(state(clearPending: true));
      confirmed = result;
      pendingKey = null;
      notice =
          'Tu pedido está confirmado. El precio de tu cotización se mantuvo.';
    } on ApiFailure catch (e) {
      failure = e;
      if (e.definitive) {
        try {
          await storage.write(state(clearPending: true));
          pendingKey = null;
          quote = null;
        } catch (_) {
          /* Keep the saved key until storage recovers. */
        }
      }
    } catch (_) {
      failure = const ApiFailure(
        'STORAGE',
        'No pudimos guardar el estado de la compra. Habilita el almacenamiento y reintenta este mismo pedido.',
      );
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> findOrder(String id) async {
    if (busy || !initialized || id.trim().isEmpty) return;
    busy = true;
    failure = null;
    notice = null;
    notifyListeners();
    try {
      final result = await api.order(scope, id.trim());
      orders = [result, ...orders.where((o) => o.id != result.id)];
      await storage.write(state());
      notice = 'Pedido consultado y disponible en tu historial.';
    } on ApiFailure catch (e) {
      failure = e;
    } catch (_) {
      failure = const ApiFailure(
        'STORAGE',
        'El pedido se consultó, pero no pudimos guardarlo en este navegador.',
      );
    } finally {
      busy = false;
      notifyListeners();
    }
  }
}
