import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'models.dart';

class ApiFailure implements Exception {
  final String code, message;
  final String? traceId;
  final int? status;
  const ApiFailure(this.code, this.message, {this.traceId, this.status});
  bool get definitive => status != null && status! >= 400 && status! < 500;
  @override
  String toString() => message;
}

class MarketplaceApi {
  final http.Client client;
  final String baseUrl;
  MarketplaceApi({
    http.Client? client,
    this.baseUrl = const String.fromEnvironment(
      'API_BASE_URL',
      defaultValue: '/api',
    ),
  }) : client = client ?? http.Client();
  Future<Json> request(
    String method,
    String path,
    Scope scope, {
    Json? body,
    String? key,
  }) async {
    try {
      final req = http.Request(method, Uri.parse('$baseUrl$path'));
      req.headers.addAll({
        ...scope.headers,
        'Content-Type': 'application/json',
        if (key != null) 'Idempotency-Key': key,
      });
      if (body != null) req.body = jsonEncode(body);
      final response = await client
          .send(req)
          .then(http.Response.fromStream)
          .timeout(const Duration(seconds: 38));
      final json = jsonDecode(response.body) as Json;
      if (response.statusCode >= 400) {
        final code = json['code'] as String? ?? 'INTERNAL_ERROR';
        throw ApiFailure(
          code,
          switch (code) {
            'CREDIT_INSUFFICIENT' =>
              'Tu crédito disponible cambió. Ajusta el carrito o paga al contado.',
            'QUOTE_INVALID' =>
              'Esta cotización venció o ya fue confirmada. Genera una nueva.',
            'IDEMPOTENCY_CONFLICT' =>
              'La solicitud no coincide con el intento original.',
            'ORDER_NOT_FOUND' =>
              'No encontramos este pedido en el país y cliente seleccionados.',
            'PRODUCT_NOT_FOUND' =>
              'Uno de los productos ya no está disponible.',
            'INVALID_REQUEST' =>
              'Revisa los productos, las cantidades y los datos de tu solicitud.',
            _ =>
              'El servicio no pudo completar la solicitud. Intenta nuevamente.',
          },
          traceId: json['traceId'],
          status: response.statusCode,
        );
      }
      return json;
    } on ApiFailure {
      rethrow;
    } on TimeoutException {
      throw const ApiFailure(
        'TIMEOUT',
        'La respuesta está tardando. Puedes reintentar de forma segura.',
      );
    } catch (_) {
      throw const ApiFailure(
        'CONNECTION',
        'No pudimos conectar con la tienda. Revisa la conexión e intenta nuevamente.',
      );
    }
  }

  Future<bool> ready() async {
    try {
      await request('GET', '/ready', const Scope());
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<Quote> quote(
    Scope scope,
    String payment,
    Map<String, int> items,
  ) async => Quote(
    await request(
      'POST',
      '/quotes',
      scope,
      body: {
        ...scope.toJson(),
        'paymentMethod': payment,
        'items': items.entries
            .where((e) => e.value > 0)
            .map((e) => {'sku': e.key, 'quantity': e.value})
            .toList(),
      },
    ),
  );
  Future<Order> confirm(Quote quote, String key) async => Order(
    await request(
      'POST',
      '/orders',
      quote.scope,
      key: key,
      body: {'quoteId': quote.id, 'customerId': quote.scope.customer},
    ),
  );
  Future<Order> order(Scope scope, String id) async =>
      Order(await request('GET', '/orders/${Uri.encodeComponent(id)}', scope));
  void close() => client.close();
}
