import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_frontend/adaptive_components.dart';
import 'package:marketplace_frontend/main.dart';
import 'package:marketplace_frontend/models.dart';
import 'package:marketplace_frontend/shop_controller.dart';
import 'package:marketplace_frontend/widgets.dart';
import 'helpers.dart';

const longName =
    'Bebida artesanal de frutas tropicales sin azúcar añadida, edición especial para comercios y distribuidores de la región';
const longSecond =
    'Agua mineral natural con gas en presentación retornable familiar para abastecimiento de tiendas de barrio';
const longOrigin =
    'CAMPAÑA-REGIONAL-DE-BENEFICIOS-POR-VOLUMEN-PARA-COMERCIOS-Y-DISTRIBUIDORES-2026';
const longCatalog = [
  Product(
    'SKU-001',
    longName,
    'Presentación de demostración con una descripción extensa y condiciones comerciales visibles completas.',
    0xFFFFF4CE,
  ),
  Product(
    'SKU-002',
    longSecond,
    'Selección para distribuidores · Envase retornable',
    0xFFE5F6F6,
  ),
];

void viewport(WidgetTester tester, Size size, {double keyboard = 0}) {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  tester.view.viewPadding = const FakeViewPadding(top: 24, bottom: 48);
  tester.view.padding = FakeViewPadding(top: 24, bottom: keyboard > 0 ? 0 : 48);
  tester.view.viewInsets = FakeViewPadding(bottom: keyboard);
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetViewPadding);
  addTearDown(tester.view.resetPadding);
  addTearDown(tester.view.resetViewInsets);
}

Future<ShopController> longShop({bool quote = true}) async {
  final api = FakeApi();
  api.result.json['adjustments'][0]['promotionId'] = longOrigin;
  api.result.json['gifts'][0]['sku'] =
      'Obsequio de edición especial para clientes frecuentes con envase retornable de tamaño familiar';
  final c = ShopController(
    api,
    MemoryStorage(),
    now: () => fixedNow,
    catalog: longCatalog,
  );
  await c.initialize();
  c.preset({'SKU-001': 12, 'SKU-002': 1});
  if (quote) await c.createQuote();
  return c;
}

void noClippedParagraphs(WidgetTester tester) {
  expect(tester.takeException(), isNull);
  for (final element in find.byType(RichText).evaluate()) {
    final paragraph = element.renderObject! as RenderParagraph;
    expect(
      paragraph.didExceedMaxLines,
      false,
      reason: paragraph.text.toPlainText(),
    );
    // Selection boxes include trailing spaces beyond a soft line break. Check
    // painted words, rather than treating invisible whitespace as clipped text.
    final boxes = RegExp(r'\S+')
        .allMatches(paragraph.text.toPlainText())
        .expand(
          (word) => paragraph.getBoxesForSelection(
            TextSelection(baseOffset: word.start, extentOffset: word.end),
          ),
        );
    for (final box in boxes) {
      expect(
        box.right,
        lessThanOrEqualTo(paragraph.size.width + 1),
        reason: paragraph.text.toPlainText(),
      );
      expect(
        box.bottom,
        lessThanOrEqualTo(paragraph.size.height + 1),
        reason: paragraph.text.toPlainText(),
      );
    }
  }
}

void main() {
  setUpAll(() async {
    await (FontLoader('MariposaMarketSans')
          ..addFont(rootBundle.load('assets/fonts/Roboto-Regular.ttf'))
          ..addFont(rootBundle.load('assets/fonts/Roboto-Bold.ttf'))
          ..addFont(rootBundle.load('assets/fonts/Roboto-Black.ttf')))
        .load();
    await (FontLoader(
      'Roboto',
    )..addFont(rootBundle.load('assets/fonts/Roboto-Regular.ttf'))).load();
    await (FontLoader(
      'MaterialIcons',
    )..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'))).load();
  });

  for (final width in [600.0, 768.0, 1024.0]) {
    for (final scale in [1.0, 1.5]) {
      testWidgets('long names and origins at $width px, text scale $scale', (
        tester,
      ) async {
        viewport(tester, Size(width, 1100));
        tester.platformDispatcher.textScaleFactorTestValue = scale;
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        final c = await longShop();
        await tester.pumpWidget(MariposaMarketApp(controller: c));
        await tester.pumpAndSettle();
        noClippedParagraphs(tester);
        await tester.ensureVisible(find.text('Ver cálculo por producto'));
        await tester.tap(find.text('Ver cálculo por producto'));
        await tester.pumpAndSettle();
        expect(find.text('$longName · 12 uds.'), findsOneWidget);
        expect(find.text('SKU-001 · $longOrigin'), findsOneWidget);
        noClippedParagraphs(tester);
        await tester.ensureVisible(find.text('Confirmar pedido'));
        await tester.pumpAndSettle();
        final button = tester.getRect(
          find.widgetWithText(FilledButton, 'Confirmar pedido'),
        );
        final nav = tester.getRect(find.byType(NavigationBar));
        expect(button.bottom, lessThanOrEqualTo(nav.top));
        expect(nav.bottom, lessThanOrEqualTo(1100 - 48));
        if (width == 768 && scale == 1) {
          await expectLater(
            find.byKey(const ValueKey('order-summary')),
            matchesGoldenFile('goldens/summary_medium_long_names.png'),
          );
        }
        await tester.pumpWidget(const SizedBox());
      });
    }
  }

  for (final size in [const Size(390, 844), const Size(768, 600)]) {
    testWidgets(
      'quantity dialog actions stay above keyboard and system bar at $size',
      (tester) async {
        viewport(tester, size);
        final c = await longShop(quote: false);
        await tester.pumpWidget(MariposaMarketApp(controller: c));
        await tester.pumpAndSettle();
        final picker = find.descendant(
          of: find.byType(QuantityPicker).first,
          matching: find.byType(TextButton),
        );
        await tester.ensureVisible(picker);
        await tester.tap(picker);
        await tester.pumpAndSettle();
        tester.view.viewInsets = const FakeViewPadding(bottom: 280);
        tester.view.padding = const FakeViewPadding(top: 24);
        await tester.pumpAndSettle();
        final apply = find.widgetWithText(FilledButton, 'Aplicar');
        final rect = tester.getRect(apply);
        expect(rect.bottom, lessThanOrEqualTo(size.height - 280));
        expect(rect.top, greaterThanOrEqualTo(24));
        noClippedParagraphs(tester);
        await expectLater(
          find.byType(Overlay).first,
          matchesGoldenFile('goldens/keyboard_${size.width.toInt()}.png'),
        );
        await tester.enterText(find.byType(TextFormField), '15');
        await tester.tap(apply);
        await tester.pumpAndSettle();
        expect(c.items['SKU-001'], 15);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      },
    );
  }

  testWidgets('checkout button remains reachable with keyboard open', (
    tester,
  ) async {
    viewport(tester, const Size(768, 960), keyboard: 320);
    final c = await longShop();
    await tester.pumpWidget(MariposaMarketApp(controller: c));
    await tester.pumpAndSettle();
    expect(find.byType(NavigationBar), findsNothing);
    await tester.ensureVisible(find.text('Confirmar pedido'));
    await tester.pumpAndSettle();
    expect(
      tester
          .getRect(find.widgetWithText(FilledButton, 'Confirmar pedido'))
          .bottom,
      lessThanOrEqualTo(640),
    );
    noClippedParagraphs(tester);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'order lookup remains actionable above keyboard on a medium screen',
    (tester) async {
      viewport(tester, const Size(768, 640));
      final c = controller();
      await c.initialize();
      await tester.pumpWidget(MariposaMarketApp(controller: c));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Mis pedidos'));
      await tester.pumpAndSettle();
      tester.view.viewInsets = const FakeViewPadding(bottom: 280);
      tester.view.padding = const FakeViewPadding(top: 24);
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField),
        sampleOrder(sampleQuote()).id,
      );
      await tester.ensureVisible(find.text('Consultar pedido'));
      await tester.pumpAndSettle();
      expect(
        tester
            .getRect(find.widgetWithText(FilledButton, 'Consultar pedido'))
            .bottom,
        lessThanOrEqualTo(360),
      );
      noClippedParagraphs(tester);
      await tester.tap(find.text('Consultar pedido'));
      await tester.pumpAndSettle();
      expect(c.orders, hasLength(1));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'media of different aspect ratios uses identical contain viewports',
    (tester) async {
      viewport(tester, const Size(768, 1024));
      await tester.pumpWidget(
        const MaterialApp(
          home: SafeAppScaffold(
            body: Padding(
              padding: EdgeInsets.all(24),
              child: ProductGrid(
                children: [
                  ProductMedia(
                    child: SizedBox(
                      key: ValueKey('wide'),
                      width: 400,
                      height: 100,
                      child: ColoredBox(color: Colors.amber),
                    ),
                  ),
                  ProductMedia(
                    child: SizedBox(
                      key: ValueKey('tall'),
                      width: 100,
                      height: 400,
                      child: ColoredBox(color: Colors.green),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final media = find.byType(ProductMedia);
      expect(tester.getSize(media.at(0)), tester.getSize(media.at(1)));
      for (final id in ['wide', 'tall']) {
        final child = tester.renderObject<RenderBox>(find.byKey(ValueKey(id)));
        final frame = tester.renderObject<RenderBox>(
          media.at(id == 'wide' ? 0 : 1),
        );
        final transform = child.getTransformTo(frame);
        expect(transform.entry(0, 0), closeTo(transform.entry(1, 1), 0.00001));
        final painted = MatrixUtils.transformRect(
          transform,
          Offset.zero & child.size,
        );
        expect(painted.left, greaterThanOrEqualTo(ProductMedia.inset - .01));
        expect(painted.top, greaterThanOrEqualTo(ProductMedia.inset - .01));
        expect(
          painted.right,
          lessThanOrEqualTo(frame.size.width - ProductMedia.inset + .01),
        );
        expect(
          painted.bottom,
          lessThanOrEqualTo(frame.size.height - ProductMedia.inset + .01),
        );
      }
      noClippedParagraphs(tester);
    },
  );
}
