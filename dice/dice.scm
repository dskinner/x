(define-module (dice)
  #:export (action action? make-action action-speed action-power))

(use-modules (ice-9 exceptions)
             (ice-9 format)
             (ice-9 receive)
             (ice-9 threads)
             (fibers)
             (fibers channels)
             (fibers conditions)
             (fibers operations)
             (srfi srfi-1)
             (srfi srfi-9)
             (srfi srfi-43))

;; returns sample space for rolls of given dice set nDs. The table is sized
;; as columns of length n and rows of length s^n. Notating each column from
;; s^0..s^n-1, this is the number of times a column value is repeatedly assigned
;; before incrementing to the next value.
;;
;; For example, column zero of 3d4 would be notated 4^0 which equals 1. So column
;; values would go 1, 2, 3, 4, 1, 2, 3, 4, 1, 2 and so on.
;;
;; Column one would be notated 4^1 which equals 4. Column values would go
;; 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, and so on.
;;
;; Column two of 3d4 would be notated 4^2 which equals 16. The first sixteen column
;; values would be 1, the next sixteen would be 2, and so on.
(define-public (gen n s)
  ;; precompute column span sizes
  (define sizes (vector-unfold (λ (j) (expt s j)) n))
  
  (vector-unfold
   (λ (i)
     (vector-unfold
      (λ (j)
        (+ 1 (modulo (truncate (/ i (vector-ref sizes j))) s)))
      n))
   (expt s n)))

;; convenience method returning all actions for sample space nDs.
(define-public (gen-actions n s)
  (vector->list
   (vector-map (λ (_ x) (vector->actions x s)) (gen n s))))

;; convenience method returning all valid actions sorted for sample space nDs.
(define-public (gen-actions-valid-sorted n s)
  (filter (negate null-list?) ;; some rolls have no valid actions
          (map (λ (a) (sort (filter action-valid? a) (negate less-speed-power)))
               (gen-actions n s))))

;; action is the core game mechanic defined by grouping dice of the same face-value.
;; There must be at least two of the same face-value to be considered a valid action.
(define-record-type action
  (make-action power speed)
  action?
  (power action-power)  ;; face-value of dice roll
  (speed action-speed)) ;; number of dice for given face-value

;; predicate identifying if at least two dice rolled the same face-value.
(define-public (action-valid? a) (<= 2 (action-speed a)))

;; an action penalty removes a die from the set, potentially making the action no longer valid.
(define-public (action-with-penalty a)
  (make-action (action-power a) (- (action-speed a) 1)))

;; while conflict resolution has many possibilities, this implementation assumes the desire to penalize
;; the action with the highest power and speed; this may not necessarily be the best move.
(define-public (action-list-penalize l)
  (sort (filter action-valid? (cons (action-with-penalty (car l)) (cdr l))) (negate less-speed-power)))

;; transforms a vector of dice rolls into a list of actions, e.g. 5d6:
;;   (vector->actions #(1 1 1 4 4) 6)
;;   (list #<action power: 1 speed: 3> #<action power: 4 speed: 2> ...)
(define-public (vector->actions v s)
  (vector->list ;; TODO consider srfi-1 for list (unfold ...)
   (vector-unfold (λ (i) (make-action (+ 1 i) (vector-count (λ (_ x) (= x (+ 1 i))) v))) s)))

;; less by speed, then power
(define-public (less-speed-power a b)
  (or 
   (< (action-speed a) (action-speed b))
   (and
    (= (action-speed a) (action-speed b))
    (< (action-power a) (action-power b)))))

;; resolve a against b with seeds x and y; a and b must be sorted and valid action lists.
(define-public (conflict-resolve a x b y)
  (cond ((and (nil? a) (nil? b)) (values x y))
        ((nil? a) (conflict-resolve a x (cdr b) (+ 1 y)))
        ((nil? b) (conflict-resolve (cdr a) (+ 1 x) b y))
        ((equal? (car a) (car b)) (conflict-resolve (cdr a) x (cdr b) y))
        ((less-speed-power (car a) (car b)) (conflict-resolve (action-list-penalize a) x (cdr b) (+ 1 y)))
        ((less-speed-power (car b) (car a)) (conflict-resolve (cdr a) (+ 1 x) (action-list-penalize b) y))
        (else (raise-exception (make-exception-with-message "unreachable case"))))) ;; hopefully

;; analyze conflict result to infer win, loss, tie, partial.
;; (define-public (conflict-analyze x y) ;; actions resulted in ...
;;   (cond ((and (= 0 x) (= 0 y)) 'tie)  ;; no penalties
;;         ((and (< 0 x) (= 0 y)) 'win)  ;; penalties against y
;;         ((and (= 0 x) (< 0 y)) 'loss) ;; penalties against x
;;         ((= 0 (- x y)) 'part) ;; equal penalties against x and y
;;         ((< 0 (- x y)) 'win)  ;; penalties favoring x over y
;;         (else 'loss)))        ;; penalties favoring y over x

;; convenience to resolve and analyze a conflict.
;; (define-public (conflict-resolve-and-analyze a b)
;;   (call-with-values (λ () (conflict-resolve a 0 b 0)) conflict-analyze))

(define-record-type result
  (make-result win tie loss part)
  result?
  (win result-win set-result-win!)
  (tie result-tie set-result-tie!)
  (loss result-loss set-result-loss!)
  (part result-part set-result-part!))

;; analyze and record action results from the perspective of x.
(define-public (result-analyze! r x y)
  (cond ((and (= 0 x) (= 0 y)) (set-result-tie! r (+ 1 (result-tie r)))) ;; no penalties
        ((and (< 0 x) (= 0 y)) (set-result-win! r (+ 1 (result-win r)))) ;; penalties against y
        ((and (= 0 x) (< 0 y)) (set-result-loss! r (+ 1 (result-loss r)))) ;; penalties against x
        ((= 0 (- x y)) (set-result-part! r (+ 1 (result-part r)))) ;; equal penalties against x and y
        ((< 0 (- x y)) (set-result-win! r (+ 1 (result-win r)))) ;; penalties favoring x over y
        (else (set-result-loss! r (+ 1 (result-loss r))))))      ;; penalties favoring y over x

;; additive merge s into r.
(define-public (result-merge! r s)
  (set-result-win! r (+ (result-win r) (result-win s)))
  (set-result-tie! r (+ (result-tie r) (result-tie s)))
  (set-result-loss! r (+ (result-loss r) (result-loss s)))
  (set-result-part! r (+ (result-part r) (result-part s))))

(define-public (result-percent-coef r) (/ 100 (+ (result-win r) (result-tie r) (result-loss r) (result-part r))))
(define-public (result-win-percent r) (* (result-percent-coef r) (result-win r)))
(define-public (result->string r)
  (let ((t (result-percent-coef r)))
    (format #f "win:~,2f% tie:~,2f% loss:~,2f% part:~,2f%"
            (* t (result-win r))
            (* t (result-tie r))
            (* t (result-loss r))
            (* t (result-part r)))))

(define-public (less-win a b) (< (result-win a) (result-win b)))

;; resolve conflicts and accumulate result for dice set.
(define-public (resolve-all as bs)
  (define r (make-result 0 0 0 0))
  (for-each
   (λ (a)
     (for-each
      (λ (b)
        (receive (x y) (conflict-resolve a 0 b 0)
          (result-analyze! r x y)))
      bs))
   as)
  r)

;; resolve conflicts and accumulate results for individual action sets of dice set.
(define-public (resolve-each as bs)
  (define rs '())
  (for-each
   (λ (a)
     (define r (make-result 0 0 0 0))
     (for-each
      (λ (b)
        (receive (x y) (conflict-resolve a 0 b 0)
          (result-analyze! r x y)))
      bs)
     (set! rs (append rs (list r))))
   as)
  rs)

(define-syntax-rule (<- ch fn)
  (wrap-operation (get-operation ch) fn))

(define-syntax-rule (<> sig fn)
  (wrap-operation (wait-operation sig) fn))

(define-syntax-rule (select ops ...)
  (perform-operation (choice-operation ops ...)))

;; TODO could also do this with a (type chan (channel condition))
;;      where for-message expands to when #t select <-work or <>done
;; (define (worker)
;;   (define r (make-result 0 0 0 0))
;;   (for-message
;;    (λ (a)
;;      (for-each
;;       (λ (b) (receive (x y) (conflict-resolve a 0 b 0) (result-analyze! r x y)))
;;       bs))
;;    work)
;;   (put-message resp r))

(define-public (resolve-all-fan-out as bs)
  (define done (make-condition))
  (define work (make-channel))
  (define resp (make-channel))

  (define (worker)
    (define r (make-result 0 0 0 0))
    (while #t
      (select
       (<- work
           (λ (a)
             (for-each
              (λ (b) (receive (x y) (conflict-resolve a 0 b 0) (result-analyze! r x y)))
              bs)))
       (<> done break)))
    (put-message resp r))

  (define nworkers 10)

  (let lp ((n nworkers))
    (spawn-fiber worker #:parallel? #t)
    (when (> n 1) (lp (- n 1))))
  
  (spawn-fiber
   (λ () (for-each (λ (a) (put-message work a)) as) (signal-condition! done))
   #:parallel? #t)
  
  (define r (make-result 0 0 0 0))
  (let lp ((n nworkers))
    (result-merge! r (get-message resp))
    (when (> n 1) (lp (- n 1))))
  r)

(define-public (resolve-each-fan-out as bs)
  (define done (make-condition))
  (define work (make-channel))
  (define resp (make-channel))

  (define (worker)
    (while #t
      (select
       (<- work
           (λ (a)
             (define r (make-result 0 0 0 0))
             (for-each
              (λ (b) (receive (x y) (conflict-resolve a 0 b 0) (result-analyze! r x y)))
              bs)
             (put-message resp r)))
       (<> done break))))

  (define nworkers 10)

  (let lp ((n nworkers))
    (spawn-fiber worker #:parallel? #t)
    (when (> n 1) (lp (- n 1))))
  
  (spawn-fiber
   (λ () (for-each (λ (a) (put-message work a)) as) (signal-condition! done))
   #:parallel? #t)
  
  (define rs '())
  (let lp ((n (length as)))
    (set! rs (append rs (list (get-message resp))))
    (when (> n 1) (lp (- n 1))))
  rs)

(define-public (resolve-all-split as bs)
  (define (split-into l n)
    (receive (a b) (split-at l (/ (length l) 2))
      (if (<= n 1)
          (append (list a) (list b))
          (append (split-into a (- n 1)) (split-into b (- n 1))))))
  
  (define resp (make-channel))
  
  (define (worker as)
    (define r (make-result 0 0 0 0))
    (for-each
     (λ (a)
       (for-each
        (λ (b) (receive (x y) (conflict-resolve a 0 b 0) (result-analyze! r x y)))
        bs))
     as)
    (put-message resp r))
  
  (define chunks (split-into as (sqrt (* 2 (current-processor-count)))))
  (for-each (λ (chunk) (spawn-fiber (λ () (worker chunk)) #:parallel? #t)) chunks)

  (define r (make-result 0 0 0 0))
  (let lp ((n (length chunks)))
    (result-merge! r (get-message resp))
    (when (> n 1) (lp (- n 1))))
  r)
