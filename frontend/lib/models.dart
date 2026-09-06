import 'dart:math';

typedef Json = Map<String, dynamic>;

/// Amounts remain decimal strings/BigInt, including in JavaScript builds.
class Money {
  final BigInt amount;
  final String currency;
  final int scale;
  Money(this.amount, this.currency, this.scale);
  factory Money.fromJson(Json json) => Money(
    BigInt.parse(json['amount'].toString()),
    json['currency'] as String,
    json['scale'] as int,
  );
  String get formatted {
    final raw = amount.abs().toString().padLeft(scale + 1, '0');
    final whole = scale == 0 ? raw : raw.substring(0, raw.length - scale);
    final grouped = whole.replaceAllMapped(
      RegExp(r'\B(?=(\d{3})+(?!\d))'),
      (_) => '.',
    );
    final decimal = scale == 0 ? '' : ',${raw.substring(raw.length - scale)}';
    return '${amount.isNegative ? '−' : ''}${currency == 'PEN' ? 'S/ ' : '\$ '}$grouped$decimal';
  }

  @override
  bool operator ==(Object other) =>
      other is Money &&
      amount == other.amount &&
      currency == other.currency &&
      scale == other.scale;
  @override
  int get hashCode => Object.hash(amount, currency, scale);
}

class Scope {
  final String tenant, country, customer;
  const Scope({
    this.tenant = 'tenant-demo',
    this.country = 'PE',
    this.customer = 'CUSTOMER-001',
  });
  Json toJson() => {
    'tenantId': tenant,
    'country': country,
    'customerId': customer,
  };
  Map<String, String> get headers => {
    'X-Tenant-ID': tenant,
    'X-Country': country,
    'X-Customer-ID': customer,
  };
  factory Scope.fromJson(Json json) => Scope(
    tenant: json['tenantId'],
    country: json['country'],
    customer: json['customerId'],
  );
  String get key => '$tenant/$country/$customer';
}

class Quote {
  final Json json;
  Quote(this.json);
  String get id => json['quoteId'];
  Scope get scope => Scope.fromJson(json);
  Money money(String field) => Money.fromJson(json[field]);
  Money get total => money('total');
  DateTime get expiresAt => DateTime.parse(json['expiresAt']);
  bool expired(DateTime now) => !now.toUtc().isBefore(expiresAt);
  String get payment => json['paymentMethod'];
  bool get eligible => json['creditEvaluation']['eligible'] == true;
  Money get available => Money.fromJson(json['creditEvaluation']['available']);
  List<Json> rows(String field) => (json[field] as List? ?? [])
      .map((e) => Map<String, dynamic>.from(e as Map))
      .toList();
}

class Order {
  final Json json;
  Order(this.json);
  String get id => json['orderId'];
  String get quoteId => json['quoteId'];
  Scope get scope => Scope.fromJson(json);
  Money get total => Money.fromJson(json['total']);
}

class Product {
  final String sku, name, subtitle;
  final int color;
  const Product(this.sku, this.name, this.subtitle, this.color);
}

// Presentation catalog for the supplied demo seed. Prices are never calculated here.
const products = [
  Product(
    'SKU-001',
    'Bebida original',
    'Línea de bebidas · Producto demo',
    0xFFFFF4CE,
  ),
  Product(
    'SKU-002',
    'Bebida selección',
    'Línea de bebidas · Producto demo',
    0xFFE5F6F6,
  ),
];
String productName(String sku) => switch (sku) {
  'SKU-001' => 'Bebida original',
  'SKU-002' => 'Bebida selección',
  'GIFT-001' => 'Obsequio de la casa',
  _ => sku,
};

String newKey() {
  final random = Random.secure();
  final bytes = List.generate(16, (_) => random.nextInt(256));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  final hex = bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  return '${hex.substring(0, 8)}-${hex.substring(8, 12)}-${hex.substring(12, 16)}-${hex.substring(16, 20)}-${hex.substring(20)}';
}
