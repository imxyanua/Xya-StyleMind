export type CartProduct = {
  id: string;
  name: string;
  price: number;
  stock: number;
  image_url: string;
  style: string;
  color: string;
};

export type CartItem = {
  id: string;
  product: CartProduct;
  quantity: number;
  subtotal: number;
};

export type Cart = {
  cart_id: string;
  user_id: string;
  items: CartItem[];
  total: number;
};
