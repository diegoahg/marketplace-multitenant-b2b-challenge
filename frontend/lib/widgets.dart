import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'models.dart';
import 'adaptive_components.dart';

const ink = Color(0xFF17212B);
const primary = Color(0xFF123B5D);
const secondary = Color(0xFF00A6A6);
const action = Color(0xFFFFC629);
const success = Color(0xFF22A06B);
const secondaryTint = Color(0xFFE5F6F6);
const actionTint = Color(0xFFFFF4CE);
const muted = Color(0xFF596875);
const canvas = Color(0xFFF5F7F8);
const border = Color(0xFFDCE3E8);

class Surface extends StatelessWidget {
  final Widget child;
  final EdgeInsets padding;
  final Color color;
  final Key? repaintKey;
  const Surface({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(24),
    this.color = Colors.white,
    this.repaintKey,
  });
  @override
  Widget build(BuildContext context) {
    final surface = Container(
      padding: padding,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: border),
      ),
      child: Material(color: Colors.transparent, child: child),
    );
    return repaintKey == null
        ? surface
        : RepaintBoundary(key: repaintKey, child: surface);
  }
}

class Tag extends StatelessWidget {
  final String label;
  final Color color;
  const Tag(this.label, {super.key, this.color = primary});
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
    decoration: BoxDecoration(
      color: color.withValues(alpha: .08),
      borderRadius: BorderRadius.circular(7),
    ),
    child: Text(
      label,
      style: TextStyle(
        color: color,
        fontSize: 11,
        fontWeight: FontWeight.w700,
        letterSpacing: .3,
      ),
    ),
  );
}

class AmountRow extends StatelessWidget {
  final String label, value;
  final bool emphasis;
  final Color? color;
  const AmountRow(
    this.label,
    this.value, {
    super.key,
    this.emphasis = false,
    this.color,
  });
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 7),
    child: AdaptivePair(
      breakpoint: 280,
      primary: Text(
        label,
        style: TextStyle(color: color ?? muted, fontSize: emphasis ? 17 : 13),
      ),
      secondary: Text(
        value,
        textAlign: TextAlign.end,
        style: TextStyle(
          color: color ?? ink,
          fontSize: emphasis ? 26 : 14,
          fontWeight: emphasis ? FontWeight.w800 : FontWeight.w600,
        ),
      ),
    ),
  );
}

class BottleArt extends StatelessWidget {
  final bool second;
  final double height;
  const BottleArt({super.key, this.second = false, this.height = 155});
  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: SizedBox(
      height: height,
      width: 180,
      child: CustomPaint(painter: _Bottles(second)),
    ),
  );
}

class _Bottles extends CustomPainter {
  final bool second;
  _Bottles(this.second);
  @override
  void paint(Canvas c, Size s) {
    c.save();
    c.translate(s.width / 2, s.height / 2);
    c.scale(s.height / 180);
    final shadow = Paint()..color = ink.withValues(alpha: .10);
    c.drawOval(const Rect.fromLTWH(-65, 63, 130, 14), shadow);
    void bottle(double x, double y, double angle, double scale, bool back) {
      c.save();
      c.translate(x, y);
      c.rotate(angle);
      c.scale(scale);
      final body = second ? secondary : action;
      final p = Paint()..color = back ? body.withValues(alpha: .65) : body;
      final path = Path()
        ..moveTo(-12, -67)
        ..lineTo(12, -67)
        ..lineTo(12, -43)
        ..cubicTo(12, -33, 30, -30, 30, -13)
        ..lineTo(30, 58)
        ..quadraticBezierTo(30, 68, 20, 68)
        ..lineTo(-20, 68)
        ..quadraticBezierTo(-30, 68, -30, 58)
        ..lineTo(-30, -13)
        ..cubicTo(-30, -30, -12, -33, -12, -43)
        ..close();
      c.drawPath(path, p);
      c.drawRRect(
        RRect.fromRectAndRadius(
          const Rect.fromLTWH(-14, -74, 28, 13),
          const Radius.circular(3),
        ),
        Paint()..color = second ? primary : primary,
      );
      c.drawRRect(
        RRect.fromRectAndRadius(
          const Rect.fromLTWH(-25, -4, 50, 47),
          const Radius.circular(2),
        ),
        Paint()..color = Colors.white,
      );
      c.drawLine(
        const Offset(-20, -18),
        const Offset(-20, 48),
        Paint()
          ..color = Colors.white.withValues(alpha: .18)
          ..strokeWidth = 5
          ..strokeCap = StrokeCap.round,
      );
      final text = TextPainter(
        text: TextSpan(
          text: 'MariposaMarket',
          style: TextStyle(
            color: second ? primary : primary,
            fontSize: 5,
            fontFamily: 'MariposaMarketSans',
            fontWeight: FontWeight.w900,
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      text.paint(c, Offset(-text.width / 2, 6));
      c.drawLine(
        const Offset(-11, 29),
        const Offset(11, 29),
        Paint()
          ..color = body
          ..strokeWidth = 2,
      );
      c.restore();
    }

    bottle(31, 2, .14, .88, true);
    bottle(-19, -2, -.10, 1, false);
    c.restore();
  }

  @override
  bool shouldRepaint(_Bottles oldDelegate) => oldDelegate.second != second;
}

class QuantityPicker extends StatelessWidget {
  final String sku;
  final String? productLabel;
  final int value;
  final bool disabled;
  final ValueChanged<int> onChanged;
  const QuantityPicker({
    super.key,
    required this.sku,
    this.productLabel,
    required this.value,
    required this.onChanged,
    this.disabled = false,
  });
  Future<void> edit(BuildContext context) async {
    var quantity = value.toString();
    final form = GlobalKey<FormState>();
    final result = await showDialog<int>(
      context: context,
      builder: (ctx) => SafeFormDialog(
        title: const Text('Elige la cantidad'),
        content: Form(
          key: form,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(productLabel ?? productName(sku), softWrap: true),
              const SizedBox(height: 16),
              TextFormField(
                initialValue: quantity,
                onChanged: (text) => quantity = text,
                autofocus: true,
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                decoration: InputDecoration(
                  label: const Text('Unidades', style: TextStyle(height: 1.5)),
                  helperText: 'De 0 a 1.000.000. Cero elimina el producto.',
                  helperMaxLines: 3,
                ),
                validator: (v) {
                  final n = int.tryParse(v ?? '');
                  return n == null || n < 0 || n > 1000000
                      ? 'Ingresa una cantidad válida.'
                      : null;
                },
                onFieldSubmitted: (_) {
                  if (form.currentState!.validate()) {
                    Navigator.pop(ctx, int.parse(quantity));
                  }
                },
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancelar'),
          ),
          FilledButton(
            onPressed: () {
              if (form.currentState!.validate()) {
                Navigator.pop(ctx, int.parse(quantity));
              }
            },
            child: const Text('Aplicar'),
          ),
        ],
      ),
    );
    if (result != null) onChanged(result);
  }

  @override
  Widget build(BuildContext context) => Container(
    decoration: BoxDecoration(
      border: Border.all(color: border),
      borderRadius: BorderRadius.circular(10),
    ),
    child: Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          tooltip: 'Quitar una unidad de $sku',
          onPressed: disabled || value == 0 ? null : () => onChanged(value - 1),
          icon: const Icon(Icons.remove, size: 17),
        ),
        TextButton(
          onPressed: disabled ? null : () => edit(context),
          style: TextButton.styleFrom(
            minimumSize: const Size(40, 44),
            padding: const EdgeInsets.symmetric(horizontal: 4),
          ),
          child: Text(
            '$value',
            semanticsLabel: 'Cantidad de $sku: $value',
            style: const TextStyle(fontWeight: FontWeight.w700, color: ink),
          ),
        ),
        IconButton(
          tooltip: 'Agregar una unidad de $sku',
          onPressed: disabled || value >= 1000000
              ? null
              : () => onChanged(value + 1),
          icon: const Icon(Icons.add, size: 17),
        ),
      ],
    ),
  );
}

class ProductTile extends StatelessWidget {
  final Product product;
  final int quantity;
  final bool locked;
  final ValueChanged<int> onChanged;
  const ProductTile({
    super.key,
    required this.product,
    required this.quantity,
    required this.locked,
    required this.onChanged,
  });
  @override
  Widget build(BuildContext context) => Surface(
    padding: const EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          decoration: BoxDecoration(
            color: Color(product.color),
            borderRadius: BorderRadius.circular(13),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(10, 10, 10, 0),
                child: Tag(
                  product.sku == 'SKU-001'
                      ? 'DESCUENTO POR VOLUMEN'
                      : 'COMBINA Y AHORRA',
                ),
              ),
              ProductMedia(
                child: BottleArt(second: product.sku == 'SKU-002', height: 180),
              ),
            ],
          ),
        ),
        const SizedBox(height: 17),
        Text(
          product.sku,
          style: const TextStyle(fontSize: 10, color: muted, letterSpacing: 1),
        ),
        const SizedBox(height: 6),
        Text(
          product.name,
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 5),
        Text(
          product.subtitle,
          style: const TextStyle(fontSize: 11, color: muted),
        ),
        const SizedBox(height: 20),
        AdaptivePair(
          breakpoint: 260,
          primary: const Text(
            'Precio al\ncotizar',
            style: TextStyle(fontSize: 12, color: muted, height: 1.5),
          ),
          secondary: QuantityPicker(
            sku: product.sku,
            productLabel: product.name,
            value: quantity,
            disabled: locked,
            onChanged: onChanged,
          ),
        ),
      ],
    ),
  );
}

class EmptyState extends StatelessWidget {
  final IconData icon;
  final String title, body;
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.body,
  });
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 25),
    child: Column(
      children: [
        Container(
          padding: const EdgeInsets.all(15),
          decoration: const BoxDecoration(
            color: canvas,
            shape: BoxShape.circle,
          ),
          child: Icon(icon, color: primary, size: 27),
        ),
        const SizedBox(height: 16),
        Text(
          title,
          textAlign: TextAlign.center,
          style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 16),
        ),
        const SizedBox(height: 8),
        Text(
          body,
          textAlign: TextAlign.center,
          style: const TextStyle(fontSize: 12, color: muted, height: 1.65),
        ),
      ],
    ),
  );
}

class HeroBanner extends StatelessWidget {
  const HeroBanner({super.key});
  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    clipBehavior: Clip.antiAlias,
    decoration: BoxDecoration(
      color: primary,
      borderRadius: BorderRadius.circular(20),
    ),
    child: Stack(
      children: [
        Positioned(
          right: -60,
          top: -70,
          child: Transform.rotate(
            angle: math.pi / 7,
            child: Container(
              width: 230,
              height: 230,
              decoration: BoxDecoration(
                border: Border.all(
                  color: Colors.white.withValues(alpha: .08),
                  width: 40,
                ),
                borderRadius: BorderRadius.circular(70),
              ),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(27),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'MÁS PARA TU NEGOCIO',
                style: TextStyle(
                  color: action,
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.8,
                ),
              ),
              const SizedBox(height: 12),
              const Text(
                'Abastece tu tienda.\nHaz rendir cada compra.',
                style: TextStyle(
                  color: Colors.white,
                  fontSize: 29,
                  height: 1.18,
                  fontWeight: FontWeight.w700,
                  letterSpacing: -.8,
                ),
              ),
              const SizedBox(height: 15),
              Text(
                'Combos, descuentos y obsequios.\nTodo claro antes de confirmar.',
                style: TextStyle(
                  color: Colors.white.withValues(alpha: .8),
                  fontSize: 13,
                  height: 1.65,
                ),
              ),
            ],
          ),
        ),
      ],
    ),
  );
}
