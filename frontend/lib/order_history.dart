import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'api.dart';
import 'models.dart';
import 'widgets.dart';

const historyDemoNotice =
    'Sección demostrativa: estos estados de ERP y notificaciones PUSH se muestran solo para dar visibilidad a la prueba técnica. No corresponden a la experiencia de un marketplace real.';

String deliveryLabel(Json? state) {
  if (state == null) return 'Sin información';
  if (state['done'] == true) return 'Confirmado';
  if (state['dead'] == true) return 'Falló · en DLQ';
  if ((state['failures'] as int? ?? 0) > 0) return 'Reintentando';
  return 'Pendiente';
}

class DeliveryBadges extends StatelessWidget {
  final Json? deliveries;
  const DeliveryBadges({super.key, this.deliveries});
  @override
  Widget build(BuildContext context) => Wrap(
    spacing: 10,
    runSpacing: 8,
    children: [
      for (final entry in {'erp': 'ERP', 'push': 'Notificaciones PUSH'}.entries)
        Chip(
          avatar: Icon(
            deliveries?[entry.key]?['done'] == true
                ? Icons.check_circle_outline
                : Icons.schedule,
            size: 18,
            color: deliveries?[entry.key]?['done'] == true ? success : muted,
          ),
          label: Text(
            '${entry.value}: ${deliveryLabel(deliveries?[entry.key])}',
            style: const TextStyle(fontSize: 11),
          ),
        ),
    ],
  );
}

class OrderHistoryCard extends StatefulWidget {
  final Order order;
  final MarketplaceApi api;
  const OrderHistoryCard({super.key, required this.order, required this.api});
  @override
  State<OrderHistoryCard> createState() => _OrderHistoryCardState();
}

class _OrderHistoryCardState extends State<OrderHistoryCard> {
  Json? details;
  String? error;
  bool loading = false;
  Timer? timer;
  @override
  void initState() {
    super.initState();
    unawaited(refresh());
    timer = Timer.periodic(const Duration(seconds: 10), (_) {
      final states = details?['deliveries'] as Map?;
      final complete =
          states != null &&
          states.length == 2 &&
          states.values.every((s) => s['done'] == true || s['dead'] == true);
      if (!complete && error == null) unawaited(refresh());
    });
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  Future<void> refresh() async {
    if (loading) return;
    setState(() {
      loading = true;
      error = null;
    });
    try {
      final result = await widget.api.orderDetails(
        widget.order.scope,
        widget.order.id,
      );
      if (mounted) setState(() => details = result);
    } on ApiFailure catch (e) {
      if (mounted) setState(() => error = e.message);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 14),
    child: Surface(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 16,
            runSpacing: 10,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              const Tag('COMPRA CONFIRMADA'),
              Text(
                widget.order.total.formatted,
                style: const TextStyle(
                  fontSize: 24,
                  fontWeight: FontWeight.w800,
                ),
              ),
              Text(widget.order.total.currency),
            ],
          ),
          const SizedBox(height: 12),
          TextButton(
            onPressed: () => showDialog<void>(
              context: context,
              builder: (_) =>
                  OrderDetailsDialog(order: widget.order, api: widget.api),
            ),
            child: Text(
              'Ver detalle del pedido ${widget.order.id}',
              style: const TextStyle(fontSize: 12),
            ),
          ),
          Text(
            '${widget.order.scope.country} · ${widget.order.scope.customer}',
            style: const TextStyle(fontSize: 12, color: muted),
          ),
          const SizedBox(height: 12),
          DeliveryBadges(deliveries: details?['deliveries']),
          if (loading) const LinearProgressIndicator(),
          if (error != null)
            Text(
              'No se pudo actualizar: $error',
              style: const TextStyle(color: muted, fontSize: 12),
            ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 12,
            children: [
              TextButton.icon(
                onPressed: loading ? null : refresh,
                icon: const Icon(Icons.refresh, size: 16),
                label: const Text('Actualizar estados'),
              ),
              TextButton.icon(
                onPressed: () =>
                    Clipboard.setData(ClipboardData(text: widget.order.id)),
                icon: const Icon(Icons.copy, size: 16),
                label: const Text('Copiar ID'),
              ),
            ],
          ),
        ],
      ),
    ),
  );
}

class OrderDetailsDialog extends StatefulWidget {
  final Order order;
  final MarketplaceApi api;
  const OrderDetailsDialog({super.key, required this.order, required this.api});
  @override
  State<OrderDetailsDialog> createState() => _OrderDetailsDialogState();
}

class _OrderDetailsDialogState extends State<OrderDetailsDialog> {
  late Future<Json> request;
  @override
  void initState() {
    super.initState();
    request = load();
  }

  Future<Json> load() =>
      widget.api.orderDetails(widget.order.scope, widget.order.id);
  @override
  Widget build(BuildContext context) => Dialog(
    backgroundColor: Colors.white,
    insetPadding: const EdgeInsets.all(16),
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 760),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                const Expanded(
                  child: Text(
                    'Detalle del pedido',
                    style: TextStyle(fontSize: 21, fontWeight: FontWeight.w700),
                  ),
                ),
                IconButton(
                  tooltip: 'Cerrar detalle',
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
            Flexible(
              child: SingleChildScrollView(
                child: FutureBuilder<Json>(
                  future: request,
                  builder: (context, snapshot) {
                    if (snapshot.connectionState != ConnectionState.done) {
                      return const Padding(
                        padding: EdgeInsets.all(30),
                        child: CircularProgressIndicator(),
                      );
                    }
                    if (snapshot.hasError) {
                      return const Text(
                        'No pudimos cargar el detalle. Actualiza para reintentar.',
                      );
                    }
                    return OrderDetailContent(details: snapshot.data!);
                  },
                ),
              ),
            ),
            TextButton.icon(
              onPressed: () => setState(() => request = load()),
              icon: const Icon(Icons.refresh),
              label: const Text('Actualizar detalle y estados'),
            ),
          ],
        ),
      ),
    ),
  );
}

class OrderDetailContent extends StatelessWidget {
  final Json details;
  const OrderDetailContent({super.key, required this.details});
  @override
  Widget build(BuildContext context) {
    final order = Order(details['order']);
    final q = Quote(details['quote']);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Tag('COMPRA CONFIRMADA'),
        const SizedBox(height: 12),
        SelectableText(
          '${order.json['orderNumber']}\nID: ${order.id}\nCotización: ${order.quoteId}',
        ),
        const SizedBox(height: 12),
        Text(
          'Fecha: ${DateTime.parse(order.json['createdAt']).toLocal()}\nPaís: ${order.scope.country} · Moneda: ${order.total.currency}\nCliente: ${order.scope.customer}\nTenant: ${order.scope.tenant}\nPago: ${q.payment == 'CREDIT' ? 'Crédito' : 'Contado'}',
        ),
        const Divider(),
        const Text(
          'Productos confirmados',
          style: TextStyle(fontWeight: FontWeight.w700),
        ),
        for (final line in q.rows('lines'))
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 10),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${line['quantity']} × ${productName(line['sku'])} · ${line['sku']}',
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                AmountRow(
                  'Precio unitario',
                  Money.fromJson(line['unitPrice']).formatted,
                ),
                AmountRow('Bruto', Money.fromJson(line['gross']).formatted),
                AmountRow(
                  'Descuento',
                  Money.fromJson(line['discount']).formatted,
                ),
                AmountRow(
                  'Base gravable',
                  Money.fromJson(line['taxableBase']).formatted,
                ),
                AmountRow('Impuesto', Money.fromJson(line['tax']).formatted),
              ],
            ),
          ),
        const Divider(),
        const Text(
          'Beneficios aplicados',
          style: TextStyle(fontWeight: FontWeight.w700),
        ),
        if (q.rows('adjustments').isEmpty && q.rows('gifts').isEmpty)
          const Text('Sin promociones en esta compra.'),
        for (final a in q.rows('adjustments'))
          Text(
            '${a['promotionId']} · ${a['sku']} · ${a['quantity']} uds. · − ${Money.fromJson(a['discountAmount']).formatted}',
          ),
        for (final gift in q.rows('gifts'))
          Text(
            'Regalo: ${gift['quantity']} × ${productName(gift['sku'])} · ${gift['promotionId']}',
          ),
        const Divider(),
        AmountRow('Subtotal', q.money('grossSubtotal').formatted),
        AmountRow('Descuentos', q.money('discountTotal').formatted),
        AmountRow('Base gravable', q.money('taxableBase').formatted),
        AmountRow('Impuestos', q.money('taxTotal').formatted),
        AmountRow('Total confirmado', order.total.formatted, emphasis: true),
        const Divider(),
        const Text(
          historyDemoNotice,
          style: TextStyle(fontSize: 12, color: muted),
        ),
        const SizedBox(height: 10),
        DeliveryBadges(deliveries: details['deliveries']),
        ExpansionTile(
          title: const Text('Datos completos de la prueba'),
          children: [
            Align(
              alignment: Alignment.centerLeft,
              child: SelectableText(
                const JsonEncoder.withIndent('  ').convert(details),
                style: const TextStyle(fontSize: 11),
              ),
            ),
          ],
        ),
      ],
    );
  }
}
