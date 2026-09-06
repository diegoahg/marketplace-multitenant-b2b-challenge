import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_frontend/main.dart';
import 'helpers.dart';

void main() {
  setUpAll(() async {
    final font = FontLoader('MariposaMarketSans')
      ..addFont(rootBundle.load('assets/fonts/Roboto-Regular.ttf'))
      ..addFont(rootBundle.load('assets/fonts/Roboto-Bold.ttf'))
      ..addFont(rootBundle.load('assets/fonts/Roboto-Black.ttf'));
    await font.load();
    await (FontLoader(
      'Roboto',
    )..addFont(rootBundle.load('assets/fonts/Roboto-Regular.ttf'))).load();
    await (FontLoader(
      'MaterialIcons',
    )..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'))).load();
  });
  for (final size in [
    const Size(1440, 1100),
    const Size(390, 844),
    const Size(768, 1024),
  ]) {
    testWidgets('responsive catalogue ${size.width}', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final c = controller();
      await c.initialize();
      await tester.pumpWidget(MariposaMarketApp(controller: c));
      await tester.pumpAndSettle();
      expect(find.text('Nuestros productos'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('goldens/catalogue_${size.width.toInt()}.png'),
      );
      await tester.pumpWidget(const SizedBox());
    });
  }
  for (final name in ['GT01', 'GT02', 'GT03', 'GT04', 'GT05']) {
    testWidgets(
      '$name displays backend golden totals without local repricing',
      (tester) async {
        tester.view.physicalSize = const Size(1440, 1400);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        final api = FakeApi()..result = goldenQuote(name);
        final c = controller(api: api);
        await c.initialize();
        c.preset({
          for (final line in api.result.rows('lines'))
            line['sku'] as String: line['quantity'] as int,
        });
        await c.createQuote();
        await tester.pumpWidget(MariposaMarketApp(controller: c));
        await tester.pumpAndSettle();
        expect(find.text(c.quote!.total.formatted), findsWidgets);
        expect(find.text(c.quote!.money('taxTotal').formatted), findsWidgets);
        final button = tester.widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Confirmar pedido'),
        );
        expect(button.onPressed == null, name == 'GT05');
        if (name == 'GT03') expect(find.text('1 × C'), findsOneWidget);
        expect(tester.takeException(), isNull);
        if (name == 'GT01') {
          await expectLater(
            find.byType(Scaffold),
            matchesGoldenFile('goldens/quote_desktop.png'),
          );
        }
        await tester.pumpWidget(const SizedBox());
      },
    );
  }
  testWidgets(
    'mobile scroll state does not collide with quote expansion state',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final c = controller();
      await c.initialize();
      c.preset({'SKU-001': 12});
      await tester.pumpWidget(MariposaMarketApp(controller: c));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Cotizar pedido'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Cotizar pedido'));
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      await tester.ensureVisible(find.text('Ver cálculo por producto'));
      await tester.tap(find.text('Ver cálculo por producto'));
      await tester.pumpAndSettle();
      expect(find.text('Precio unitario'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets('mobile checkout is scrollable and confirms the snapshot', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final c = controller();
    await c.initialize();
    c.preset({'SKU-001': 12});
    await c.createQuote();
    await tester.pumpWidget(MariposaMarketApp(controller: c));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Confirmar pedido'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await expectLater(
      find.byType(Scaffold),
      matchesGoldenFile('goldens/quote_mobile.png'),
    );
    await tester.tap(find.text('Confirmar pedido'));
    await tester.pumpAndSettle();
    expect(c.confirmed!.total, c.quote!.total);
    expect(find.text('PEDIDO CONFIRMADO'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets('product search has an honest empty state', (tester) async {
    final c = controller();
    await c.initialize();
    await tester.pumpWidget(MariposaMarketApp(controller: c));
    await tester.enterText(find.byType(TextField).first, 'no-existe');
    await tester.pump();
    expect(find.text('No encontramos productos'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  });
}
