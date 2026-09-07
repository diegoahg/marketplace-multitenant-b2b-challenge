import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'api.dart';
import 'models.dart';
import 'shop_controller.dart';
import 'store.dart';
import 'widgets.dart';
import 'adaptive_components.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final controller = ShopController(MarketplaceApi(), LocalCheckoutStorage());
  runApp(MariposaMarketApp(controller: controller));
  unawaited(controller.initialize());
}

class MariposaMarketApp extends StatelessWidget {
  final ShopController controller;
  const MariposaMarketApp({super.key, required this.controller});
  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'MariposaMarket · Marketplace B2B',
    debugShowCheckedModeBanner: false,
    theme: ThemeData(
      useMaterial3: true,
      fontFamily: 'MariposaMarketSans',
      scaffoldBackgroundColor: canvas,
      colorScheme: ColorScheme.fromSeed(
        seedColor: primary,
        primary: primary,
        onPrimary: Colors.white,
        secondary: secondary,
        onSecondary: ink,
        secondaryContainer: secondaryTint,
        onSecondaryContainer: primary,
        tertiary: action,
        onTertiary: ink,
        surface: Colors.white,
        onSurface: ink,
        surfaceTint: Colors.transparent,
      ),
      textTheme: ThemeData.light().textTheme.apply(
        fontFamily: 'MariposaMarketSans',
        bodyColor: ink,
        displayColor: ink,
      ),
      dividerTheme: const DividerThemeData(color: border, space: 28),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: action,
          foregroundColor: ink,
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 20),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          textStyle: const TextStyle(
            fontFamily: 'MariposaMarketSans',
            fontWeight: FontWeight.w700,
            fontSize: 13,
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
          side: const BorderSide(color: border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        labelStyle: const TextStyle(height: 1.5),
        floatingLabelStyle: const TextStyle(height: 1.5),
        filled: true,
        fillColor: canvas,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: border),
        ),
      ),
    ),
    home: ShopPage(controller: controller),
  );
}

class ShopPage extends StatefulWidget {
  final ShopController controller;
  const ShopPage({super.key, required this.controller});
  @override
  State<ShopPage> createState() => _ShopPageState();
}

class _ShopPageState extends State<ShopPage> {
  ShopController get shop => widget.controller;
  int page = 0;
  String search = '';
  final orderInput = TextEditingController();
  final basketKey = GlobalKey();
  Timer? timer;
  @override
  void initState() {
    super.initState();
    timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted && shop.quote != null) setState(() {});
    });
  }

  @override
  void dispose() {
    timer?.cancel();
    orderInput.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: shop,
    builder: (context, _) => LayoutBuilder(
      builder: (context, size) {
        final desktop =
            size.maxWidth >= 1100 &&
            size.maxHeight -
                    MediaQuery.viewInsetsOf(context).bottom -
                    MediaQuery.viewPaddingOf(context).vertical >=
                650 &&
            MediaQuery.textScalerOf(context).scale(14) <= 18;
        return SafeAppScaffold(
          body: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (desktop) sidebar(),
              Expanded(
                child: Column(
                  children: [
                    header(desktop),
                    Expanded(
                      child: SingleChildScrollView(
                        key: const PageStorageKey('shop-scroll'),
                        padding: EdgeInsets.all(size.maxWidth < 600 ? 18 : 32),
                        child: Center(
                          child: ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 1220),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                if (!shop.initialized)
                                  const LinearProgressIndicator(),
                                if (shop.failure != null)
                                  feedback(shop.failure!.message, error: true),
                                if (shop.notice != null) feedback(shop.notice!),
                                if (page == 0) catalogue() else history(),
                                const SizedBox(height: 30),
                                const Text(
                                  'ENTORNO DEMO  ·  Precios e impuestos sintéticos para la prueba técnica.',
                                  style: TextStyle(
                                    color: muted,
                                    fontSize: 10,
                                    letterSpacing: .3,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          navigation: desktop
              ? null
              : NavigationBar(
                  selectedIndex: page,
                  onDestinationSelected: (value) =>
                      setState(() => page = value),
                  backgroundColor: Colors.white,
                  destinations: const [
                    NavigationDestination(
                      icon: Icon(Icons.storefront_outlined),
                      label: 'Catálogo',
                    ),
                    NavigationDestination(
                      icon: Icon(Icons.receipt_long_outlined),
                      label: 'Mis pedidos',
                    ),
                  ],
                ),
        );
      },
    ),
  );

  Widget sidebar() => Container(
    width: 208,
    decoration: const BoxDecoration(
      color: Colors.white,
      border: Border(right: BorderSide(color: border)),
    ),
    padding: const EdgeInsets.fromLTRB(22, 30, 22, 24),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'MariposaMarket',
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.w900,
            color: primary,
            letterSpacing: -.6,
          ),
        ),
        const SizedBox(height: 4),
        const Text(
          'TU NEGOCIO, BIEN ABASTECIDO',
          style: TextStyle(fontSize: 8, letterSpacing: 1.2, color: muted),
        ),
        const SizedBox(height: 49),
        const Padding(
          padding: EdgeInsets.only(left: 12, bottom: 15),
          child: Text(
            'MI ESPACIO',
            style: TextStyle(fontSize: 10, color: muted, letterSpacing: 1.6),
          ),
        ),
        navItem(0, Icons.storefront_outlined, 'Catálogo'),
        const SizedBox(height: 8),
        navItem(1, Icons.receipt_long_outlined, 'Mis pedidos'),
        const Spacer(),
        Container(
          padding: const EdgeInsets.all(15),
          decoration: BoxDecoration(
            color: canvas,
            borderRadius: BorderRadius.circular(14),
          ),
          child: const Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(Icons.verified_user_outlined, color: primary, size: 23),
              SizedBox(height: 11),
              Text(
                'Compra con confianza',
                style: TextStyle(fontWeight: FontWeight.w700, fontSize: 12),
              ),
              SizedBox(height: 6),
              Text(
                'El precio que cotizas es el precio que confirmas.',
                style: TextStyle(fontSize: 11, height: 1.6, color: muted),
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        const Row(
          children: [
            CircleAvatar(
              radius: 16,
              backgroundColor: secondaryTint,
              child: Text(
                'TD',
                style: TextStyle(
                  color: primary,
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            SizedBox(width: 10),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Tienda demo',
                  style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
                ),
                Text(
                  'CUSTOMER-001',
                  style: TextStyle(fontSize: 9, color: muted),
                ),
              ],
            ),
          ],
        ),
      ],
    ),
  );
  Widget navItem(int index, IconData icon, String text) => Material(
    color: page == index ? secondaryTint : Colors.transparent,
    borderRadius: BorderRadius.circular(10),
    child: InkWell(
      borderRadius: BorderRadius.circular(10),
      onTap: () => setState(() => page = index),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 15),
        child: Row(
          children: [
            Icon(icon, size: 20, color: page == index ? primary : muted),
            const SizedBox(width: 12),
            Text(
              text,
              style: TextStyle(
                fontSize: 13,
                color: page == index ? primary : muted,
                fontWeight: page == index ? FontWeight.w700 : FontWeight.w500,
              ),
            ),
          ],
        ),
      ),
    ),
  );

  Widget header(bool desktop) => Container(
    padding: EdgeInsets.symmetric(horizontal: desktop ? 32 : 18, vertical: 15),
    decoration: const BoxDecoration(
      color: Colors.white,
      border: Border(bottom: BorderSide(color: border)),
    ),
    child: Row(
      children: [
        Expanded(
          child: Text(
            desktop
                ? 'Marketplace / ${page == 0 ? 'Catálogo' : 'Mis pedidos'}'
                : 'MariposaMarket',
            style: TextStyle(
              fontSize: desktop ? 12 : 18,
              color: desktop ? muted : primary,
              fontWeight: desktop ? FontWeight.w500 : FontWeight.w900,
              letterSpacing: desktop ? 0 : -.6,
            ),
          ),
        ),
        if (!desktop && page == 0)
          IconButton(
            tooltip: 'Ir a tu pedido',
            onPressed: () {
              final target = basketKey.currentContext;
              if (target != null) {
                Scrollable.ensureVisible(
                  target,
                  duration: const Duration(milliseconds: 300),
                );
              }
            },
            icon: Badge(
              label: Text('${shop.units}'),
              isLabelVisible: shop.units > 0,
              child: const Icon(
                Icons.shopping_bag_outlined,
                color: primary,
                size: 20,
              ),
            ),
          ),
        Tooltip(
          message: shop.online == true
              ? 'Servicio disponible. Pulsa para comprobar.'
              : 'Comprobar conexión con la tienda',
          child: IconButton(
            onPressed: shop.checking ? null : shop.checkHealth,
            icon: Icon(
              shop.online == true
                  ? Icons.cloud_done_outlined
                  : Icons.cloud_off_outlined,
              size: 20,
              color: shop.online == true ? success : muted,
            ),
          ),
        ),
        const SizedBox(width: 8),
        DropdownButtonHideUnderline(
          child: DropdownButton<String>(
            value: shop.scope.country,
            borderRadius: BorderRadius.circular(12),
            style: const TextStyle(
              fontFamily: 'MariposaMarketSans',
              color: ink,
              fontSize: 13,
              fontWeight: FontWeight.w600,
            ),
            items: const [
              DropdownMenuItem(value: 'PE', child: Text('Perú · PEN')),
              DropdownMenuItem(value: 'CL', child: Text('Chile · CLP')),
              DropdownMenuItem(value: 'CO', child: Text('Colombia · COP')),
              DropdownMenuItem(value: 'EC', child: Text('Ecuador · USD')),
              DropdownMenuItem(value: 'GT', child: Text('Guatemala · GTQ')),
              DropdownMenuItem(value: 'AR', child: Text('Argentina · ARS')),
            ],
            onChanged: shop.locked ? null : (v) => shop.setCountry(v!),
          ),
        ),
      ],
    ),
  );

  Widget feedback(String message, {bool error = false}) => Padding(
    padding: const EdgeInsets.only(bottom: 18),
    child: Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: error ? const Color(0xFFFFF1E9) : secondaryTint,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            error ? Icons.info_outline : Icons.check_circle_outline,
            size: 20,
            color: error ? const Color(0xFF9A4828) : success,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Semantics(
              liveRegion: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    message,
                    style: const TextStyle(fontSize: 13, height: 1.5),
                  ),
                  if (error && shop.failure?.traceId != null)
                    SelectableText(
                      'Referencia: ${shop.failure!.traceId}',
                      style: const TextStyle(fontSize: 10, color: muted),
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    ),
  );

  Widget catalogue() => LayoutBuilder(
    builder: (context, constraints) {
      final wide = constraints.maxWidth >= 850;
      final catalog = Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Un buen día empieza\ncon tu tienda lista.',
            style: TextStyle(
              fontSize: 31,
              fontWeight: FontWeight.w700,
              letterSpacing: -1,
              height: 1.2,
            ),
          ),
          const SizedBox(height: 10),
          const Text(
            'Elige tus productos. Nosotros hacemos las cuentas.',
            style: TextStyle(fontSize: 13, color: muted, height: 1.6),
          ),
          const SizedBox(height: 25),
          const HeroBanner(),
          const SizedBox(height: 28),
          Row(
            children: [
              const Expanded(
                child: Text(
                  'Nuestros productos',
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w700,
                    letterSpacing: -.4,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Tag('${shop.catalog.length} PRODUCTOS'),
            ],
          ),
          const SizedBox(height: 14),
          TextField(
            onChanged: (v) => setState(() => search = v),
            decoration: const InputDecoration(
              hintText: 'Buscar por nombre o SKU',
              prefixIcon: Icon(Icons.search, size: 20),
              isDense: true,
              fillColor: Colors.white,
            ),
          ),
          const SizedBox(height: 17),
          LayoutBuilder(
            builder: (context, box) {
              final filtered = shop.catalog
                  .where(
                    (p) => '${p.name} ${p.sku}'.toLowerCase().contains(
                      search.toLowerCase(),
                    ),
                  )
                  .toList();
              if (filtered.isEmpty) {
                return const EmptyState(
                  icon: Icons.search_off,
                  title: 'No encontramos productos',
                  body: 'Prueba con otro nombre o código de producto.',
                );
              }
              return ProductGrid(
                children: filtered
                    .map(
                      (p) => ProductTile(
                        product: p,
                        quantity: shop.items[p.sku] ?? 0,
                        locked: shop.locked,
                        onChanged: (n) => shop.quantity(p.sku, n),
                      ),
                    )
                    .toList(),
              );
            },
          ),
          const SizedBox(height: 23),
          const Text(
            '¿Por dónde empezar?',
            style: TextStyle(fontWeight: FontWeight.w700, fontSize: 14),
          ),
          const SizedBox(height: 7),
          const Text(
            'Carga una selección y descubre sus beneficios al cotizar.',
            style: TextStyle(fontSize: 12, color: muted),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              preset('Por volumen · 12 uds.', {'SKU-001': 12}),
              preset('Combo + volumen', {'SKU-001': 8, 'SKU-002': 1}),
              preset('Con obsequio · 6 uds.', {'SKU-001': 6}),
            ],
          ),
          if (shop.quote != null) ...[
            const SizedBox(height: 26),
            breakdown(shop.quote!),
          ],
        ],
      );
      if (wide) {
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: catalog),
            const SizedBox(width: 25),
            SizedBox(width: 330, child: basket()),
          ],
        );
      }
      return Column(children: [catalog, const SizedBox(height: 25), basket()]);
    },
  );

  Widget preset(String label, Map<String, int> items) => OutlinedButton(
    onPressed: shop.locked ? null : () => shop.preset(items),
    style: OutlinedButton.styleFrom(
      backgroundColor: Colors.white,
      padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 15),
      textStyle: const TextStyle(
        fontFamily: 'MariposaMarketSans',
        fontSize: 11,
      ),
    ),
    child: Text(label),
  );

  Widget basket() {
    final q = shop.quote;
    final hasOrder = shop.confirmed != null;
    final pending = shop.pendingKey != null;
    return Surface(
      key: basketKey,
      repaintKey: const ValueKey('order-summary'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.shopping_bag_outlined, size: 21, color: primary),
              const SizedBox(width: 9),
              const Text(
                'Tu pedido',
                style: TextStyle(fontSize: 19, fontWeight: FontWeight.w700),
              ),
              const Spacer(),
              Tag('${shop.units} uds.'),
            ],
          ),
          const SizedBox(height: 8),
          const Text(
            'Revisa, cotiza y confirma.',
            style: TextStyle(fontSize: 12, color: muted),
          ),
          const Divider(),
          if (shop.items.isEmpty)
            const EmptyState(
              icon: Icons.shopping_basket_outlined,
              title: 'Aquí empieza tu compra',
              body: 'Agrega productos del catálogo\npara preparar tu pedido.',
            )
          else ...[
            ...shop.items.entries.map(
              (e) => Padding(
                padding: const EdgeInsets.only(bottom: 15),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 34,
                      height: 38,
                      decoration: BoxDecoration(
                        color: e.key == 'SKU-001' ? actionTint : secondaryTint,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: const Icon(
                        Icons.local_drink_outlined,
                        size: 19,
                        color: primary,
                      ),
                    ),
                    const SizedBox(width: 11),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            shop.nameOf(e.key),
                            style: const TextStyle(
                              fontWeight: FontWeight.w600,
                              fontSize: 12,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            '${e.value} unidades · ${e.key}',
                            style: const TextStyle(fontSize: 11, color: muted),
                          ),
                        ],
                      ),
                    ),
                    IconButton(
                      tooltip: 'Eliminar ${e.key}',
                      onPressed: shop.locked
                          ? null
                          : () => shop.quantity(e.key, 0),
                      visualDensity: VisualDensity.compact,
                      icon: const Icon(Icons.close, size: 15),
                    ),
                  ],
                ),
              ),
            ),
            const Divider(),
            const Text(
              '¿Cómo prefieres pagar?',
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: SegmentedButton<String>(
                segments: const [
                  ButtonSegment(
                    value: 'CREDIT',
                    label: Text('Crédito', style: TextStyle(fontSize: 12)),
                    icon: Icon(Icons.account_balance_wallet_outlined, size: 17),
                  ),
                  ButtonSegment(
                    value: 'CASH',
                    label: Text('Contado', style: TextStyle(fontSize: 12)),
                    icon: Icon(Icons.payments_outlined, size: 17),
                  ),
                ],
                selected: {shop.payment},
                onSelectionChanged: shop.locked
                    ? null
                    : (s) => shop.setPayment(s.first),
                showSelectedIcon: false,
                style: SegmentedButton.styleFrom(
                  side: const BorderSide(color: border),
                  selectedBackgroundColor: secondaryTint,
                  selectedForegroundColor: primary,
                ),
              ),
            ),
            const SizedBox(height: 19),
            if (q == null) ...[
              const Text(
                'Verás el precio final, tus descuentos y el crédito disponible al cotizar.',
                style: TextStyle(color: muted, fontSize: 12, height: 1.65),
              ),
              const SizedBox(height: 19),
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  onPressed: shop.locked ? null : shop.createQuote,
                  icon: shop.busy
                      ? spinner()
                      : const Icon(Icons.arrow_forward, size: 18),
                  label: Text(shop.busy ? 'Cotizando…' : 'Cotizar pedido'),
                ),
              ),
            ] else ...[
              AmountRow('Subtotal', q.money('grossSubtotal').formatted),
              AmountRow(
                'Descuentos',
                '− ${q.money('discountTotal').formatted}',
                color: primary,
              ),
              AmountRow('Base gravable', q.money('taxableBase').formatted),
              AmountRow('Impuestos', q.money('taxTotal').formatted),
              const Divider(),
              AmountRow('Total', q.total.formatted, emphasis: true),
              Align(
                alignment: Alignment.centerRight,
                child: Text(
                  '${q.total.currency} · Impuestos incluidos',
                  style: const TextStyle(fontSize: 10, color: muted),
                ),
              ),
              const SizedBox(height: 18),
              if (q.payment == 'CREDIT')
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(13),
                  decoration: BoxDecoration(
                    color: q.eligible ? canvas : const Color(0xFFFFF1E9),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        q.eligible
                            ? 'Crédito disponible al cotizar'
                            : 'Crédito insuficiente',
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w700,
                          color: q.eligible ? primary : const Color(0xFF9A4828),
                        ),
                      ),
                      const SizedBox(height: 5),
                      Text(
                        q.available.formatted,
                        style: const TextStyle(
                          fontWeight: FontWeight.w700,
                          fontSize: 18,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        q.eligible
                            ? 'El saldo se vuelve a verificar al confirmar.'
                            : 'Reduce cantidades o cambia a contado.',
                        style: const TextStyle(
                          fontSize: 10,
                          color: muted,
                          height: 1.5,
                        ),
                      ),
                    ],
                  ),
                ),
              const SizedBox(height: 16),
              if (hasOrder) ...[
                const Tag('PEDIDO CONFIRMADO'),
                const SizedBox(height: 12),
                SelectableText(
                  shop.confirmed!.id,
                  style: const TextStyle(fontSize: 10, color: muted),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => setState(() => page = 1),
                    child: const Text('Ver mis pedidos'),
                  ),
                ),
                SizedBox(
                  width: double.infinity,
                  child: TextButton(
                    onPressed: () => shop.preset({}),
                    child: const Text('Preparar otro pedido'),
                  ),
                ),
              ] else ...[
                Text(
                  pending ? 'Confirmación pendiente de respuesta' : expiry(q),
                  style: TextStyle(
                    fontSize: 11,
                    color: q.expired(shop.now()) && !pending
                        ? const Color(0xFF9A4828)
                        : muted,
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: shop.canConfirm ? shop.confirm : null,
                    icon: shop.busy
                        ? spinner()
                        : const Icon(Icons.lock_outline, size: 16),
                    label: Text(
                      shop.busy
                          ? 'Confirmando…'
                          : pending
                          ? 'Reintentar confirmación'
                          : 'Confirmar pedido',
                    ),
                  ),
                ),
                if (!pending)
                  SizedBox(
                    width: double.infinity,
                    child: TextButton(
                      onPressed: shop.locked ? null : shop.createQuote,
                      child: const Text('Actualizar cotización'),
                    ),
                  ),
              ],
            ],
          ],
          const SizedBox(height: 18),
          const Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(Icons.shield_outlined, color: muted, size: 15),
              SizedBox(width: 7),
              Expanded(
                child: Text(
                  'Tu compra se confirma una sola vez, incluso si necesitas reintentar.',
                  style: TextStyle(color: muted, fontSize: 10, height: 1.55),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget spinner() => const SizedBox(
    width: 16,
    height: 16,
    child: CircularProgressIndicator(strokeWidth: 2),
  );
  String expiry(Quote q) {
    final seconds = q.expiresAt.difference(shop.now()).inSeconds;
    return seconds <= 0
        ? 'Cotización vencida. Actualízala para continuar.'
        : 'Precio reservado por ${seconds ~/ 60}:${(seconds % 60).toString().padLeft(2, '0')}';
  }

  Widget breakdown(Quote q) => Surface(
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Row(
          children: [
            Icon(Icons.auto_awesome_outlined, color: primary, size: 20),
            SizedBox(width: 10),
            Expanded(
              child: Text(
                'Cada beneficio, a la vista',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
              ),
            ),
          ],
        ),
        const SizedBox(height: 9),
        const Text(
          'Primero combos, luego descuentos por volumen y obsequios. Los impuestos se aplican al final.',
          style: TextStyle(fontSize: 12, color: muted, height: 1.6),
        ),
        const Divider(),
        if (q.rows('adjustments').isEmpty)
          const Text(
            'Esta selección no tiene descuentos aplicados.',
            style: TextStyle(fontSize: 12, color: muted),
          ),
        ...q
            .rows('adjustments')
            .map(
              (a) => Padding(
                padding: const EdgeInsets.only(bottom: 13),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(
                      a['promotionType'] == 'COMBO'
                          ? Icons.inventory_2_outlined
                          : Icons.trending_down,
                      color: primary,
                      size: 20,
                    ),
                    const SizedBox(width: 11),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '${a['promotionType'] == 'COMBO' ? 'Precio de combo' : 'Descuento por volumen'} · ${a['quantity']} uds.',
                            style: const TextStyle(
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            '${a['sku']} · ${a['promotionId']}',
                            style: const TextStyle(fontSize: 10, color: muted),
                          ),
                        ],
                      ),
                    ),
                    Text(
                      '− ${Money.fromJson(a['discountAmount']).formatted}',
                      style: const TextStyle(
                        color: primary,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                ),
              ),
            ),
        ...q
            .rows('gifts')
            .map(
              (g) => Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: canvas,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Icon(
                        Icons.card_giftcard_outlined,
                        color: primary,
                        size: 22,
                      ),
                      const SizedBox(width: 11),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '${g['quantity']} × ${shop.nameOf(g['sku'])}',
                              style: const TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              '${g['sku']} · Campaña ${g['promotionId']}',
                              style: const TextStyle(
                                fontSize: 10,
                                color: muted,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              'Valor de referencia: ${Money.fromJson(g['referenceValue']).formatted}\nImpuesto del obsequio: ${Money.fromJson(g['tax']).formatted}',
                              style: const TextStyle(
                                fontSize: 10,
                                color: muted,
                                height: 1.5,
                              ),
                            ),
                          ],
                        ),
                      ),
                      const Tag('REGALO'),
                    ],
                  ),
                ),
              ),
            ),
        const SizedBox(height: 12),
        ExpansionTile(
          key: PageStorageKey('quote-detail-${q.id}'),
          tilePadding: EdgeInsets.zero,
          title: const Text(
            'Ver cálculo por producto',
            style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
          ),
          children: q
              .rows('lines')
              .map(
                (line) => Padding(
                  padding: const EdgeInsets.only(bottom: 18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        '${shop.nameOf(line['sku'])} · ${line['quantity']} uds.',
                        style: const TextStyle(
                          fontWeight: FontWeight.w700,
                          fontSize: 12,
                        ),
                      ),
                      Text(
                        '${line['comboQuantity']} unidades en combo',
                        style: const TextStyle(fontSize: 10, color: muted),
                      ),
                      AmountRow(
                        'Precio unitario',
                        Money.fromJson(line['unitPrice']).formatted,
                      ),
                      AmountRow(
                        'Bruto',
                        Money.fromJson(line['gross']).formatted,
                      ),
                      AmountRow(
                        'Descuento',
                        Money.fromJson(line['discount']).formatted,
                      ),
                      AmountRow(
                        'Base gravable',
                        Money.fromJson(line['taxableBase']).formatted,
                      ),
                      AmountRow(
                        'Impuesto',
                        Money.fromJson(line['tax']).formatted,
                      ),
                    ],
                  ),
                ),
              )
              .toList(),
        ),
        const SizedBox(height: 8),
        SelectableText(
          'Cotización ${q.id}',
          style: const TextStyle(fontSize: 10, color: muted),
        ),
      ],
    ),
  );

  Widget history() => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      const Text(
        'Tus pedidos, en un lugar.',
        style: TextStyle(
          fontSize: 30,
          fontWeight: FontWeight.w700,
          letterSpacing: -.8,
        ),
      ),
      const SizedBox(height: 10),
      const Text(
        'Consulta una compra y vuelve a su precio confirmado.',
        style: TextStyle(fontSize: 13, color: muted),
      ),
      const SizedBox(height: 25),
      Surface(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Buscar un pedido',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 9),
            const Text(
              'Usa el identificador de tu confirmación. Se consulta en el país seleccionado.',
              style: TextStyle(fontSize: 12, color: muted, height: 1.5),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: orderInput,
              onSubmitted: shop.busy ? null : shop.findOrder,
              decoration: const InputDecoration(
                label: Text(
                  'Identificador del pedido',
                  style: TextStyle(height: 1.5),
                ),
                hintText: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx',
                prefixIcon: Icon(Icons.search),
              ),
            ),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: shop.busy
                  ? null
                  : () => shop.findOrder(orderInput.text),
              child: Text(shop.busy ? 'Consultando…' : 'Consultar pedido'),
            ),
          ],
        ),
      ),
      const SizedBox(height: 25),
      Row(
        children: [
          const Text(
            'Historial de este navegador',
            style: TextStyle(fontWeight: FontWeight.w700, fontSize: 17),
          ),
          const Spacer(),
          Tag('${shop.scopedOrders.length}'),
        ],
      ),
      const SizedBox(height: 14),
      if (shop.scopedOrders.isEmpty)
        const Surface(
          child: Center(
            child: EmptyState(
              icon: Icons.receipt_long_outlined,
              title: 'Tu próximo pedido empieza aquí',
              body:
                  'Las compras que confirmes aparecerán en este navegador.\nTambién puedes buscar una compra por su identificador.',
            ),
          ),
        ),
      ...shop.scopedOrders.map(
        (o) => Padding(
          padding: const EdgeInsets.only(bottom: 14),
          child: Surface(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Wrap(
                  spacing: 20,
                  runSpacing: 12,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    const Tag('CONFIRMADO'),
                    Text(
                      o.total.formatted,
                      style: const TextStyle(
                        fontSize: 24,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    Text(
                      o.total.currency,
                      style: const TextStyle(fontSize: 12, color: muted),
                    ),
                  ],
                ),
                const SizedBox(height: 13),
                Text(
                  o.id,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 12,
                  ),
                ),
                const SizedBox(height: 7),
                Text(
                  '${o.scope.country} · ${o.scope.customer}',
                  style: const TextStyle(fontSize: 11, color: muted),
                ),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 10,
                  children: [
                    OutlinedButton.icon(
                      onPressed: () async {
                        await Clipboard.setData(ClipboardData(text: o.id));
                        if (mounted) {
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(
                              content: Text('Identificador copiado'),
                            ),
                          );
                        }
                      },
                      icon: const Icon(Icons.copy, size: 15),
                      label: const Text('Copiar ID'),
                    ),
                    TextButton(
                      onPressed: shop.busy ? null : () => shop.findOrder(o.id),
                      child: const Text('Consultar de nuevo'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    ],
  );
}
