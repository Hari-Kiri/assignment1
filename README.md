# 2 types of users (buyers, and sellers):
URL: http://localhost/login
POST  data: {"account":{"user":"user_name","password":"user_password"}} in base64 encoded

# User sellers can list his merchs & User seller can update his merchs quantity
URL: http://localhost/merchs
POST data: {"account":{"user":"user_name","password":"user_password"}} in base64 encoded

# User seller can monitor his merchs quantity
URL: http://localhost/merchsupdate
POST data: {"account":{"user":"user_name","password":"user_password"},"update":{"merchsId":merchs_id_int,"quantity":merchs_quantity}} in base64 encoded

# User buyers can see list of merchs
URL: http://localhost/allmerchs
POST data: {"account":{"user":"user_name","password":"user_password"},"update":{"merchsId":merchs_id_int,"quantity":merchs_quantity_int}} in base64 encoded

# User buyers can make a purchase
URL: http://localhost/purchase
POST data: {"account":{"user":"user_name","password":"user_password"},"purchase":{"merchsId":merchs_id_int,"purchaseItem":"merchs_name","sellerId":seller_id_int,"quantity":purchase_quantity_int}} in base64 encode

# Example API consume in PHP
public function Api($payload) {
    $ch = curl_init('http://localhost/purchase');
    curl_setopt($ch, CURLOPT_POST, 1);
    curl_setopt($ch, CURLOPT_POSTFIELDS, base64_encode($payload));
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
    curl_setopt($ch, CURLOPT_HEADER, 1);
    curl_setopt($ch, CURLOPT_HTTPHEADER, array('Content-Type: text/plain'));
    $result = curl_exec($ch);
    curl_close($ch);
    list($headers, $content) = explode("\r\n\r\n", $result, 2);
    foreach (explode("\r\n", $headers) as $header) echo $header.'<br>';
    echo 'Payload: '.$payload.'<br>';
    echo 'Hashed payload: '.base64_encode($payload).'<br>';
    echo 'Encoded response: '.$content.'<br>';
    echo 'Response: '.base64_decode($content).'<br>';
    return;
}
