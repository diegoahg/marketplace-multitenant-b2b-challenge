param([string]$BaseUrl = 'http://localhost:8080')
$ErrorActionPreference = 'Stop'
$headers = @{ 'X-Tenant-ID'='tenant-demo'; 'X-Country'='PE'; 'X-Customer-ID'='CUSTOMER-001' }
$cart = @{ tenantId='tenant-demo'; country='PE'; customerId='CUSTOMER-001'; paymentMethod='CREDIT'; items=@(@{sku='SKU-001';quantity=12}) } | ConvertTo-Json -Depth 5
$quote = Invoke-RestMethod -Method Post -Uri "$BaseUrl/quotes" -Headers $headers -ContentType 'application/json' -Body $cart
$headers['Idempotency-Key'] = [guid]::NewGuid().ToString()
$request = @{quoteId=$quote.quoteId;customerId='CUSTOMER-001'} | ConvertTo-Json
$order = Invoke-RestMethod -Method Post -Uri "$BaseUrl/orders" -Headers $headers -ContentType 'application/json' -Body $request
$retry = Invoke-RestMethod -Method Post -Uri "$BaseUrl/orders" -Headers $headers -ContentType 'application/json' -Body $request
$stored = Invoke-RestMethod -Uri "$BaseUrl/orders/$($order.orderId)" -Headers $headers
if ($quote.total.amount -ne $order.total.amount -or $order.orderId -ne $retry.orderId -or $stored.orderId -ne $order.orderId) { throw 'Snapshot/idempotency/read-after-write check failed' }
$stored | ConvertTo-Json -Depth 5
Write-Output 'PASS: quote total preserved, idempotency replay, order immediately readable.'
