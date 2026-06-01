export type OrderProduct = {
  id: string;
  name: string;
  image_url: string;
  style: string;
  color: string;
};

export type OrderItem = {
  id: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  product: OrderProduct;
};

export type Order = {
  id: string;
  user_id: string;
  status: "pending" | "paid" | "shipping" | "completed" | "cancelled";
  total_amount: number;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
};
